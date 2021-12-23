package terraform

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"log"
	"strings"
)

type InputValue struct {
	Value      cty.Value
	SourceType ValueSourceType
	// SourceRange provides source location information for values whose
	// SourceType is either ValueFromConfig or ValueFromFile. It is not
	// populated for other source types, and so should not be used.
	SourceRange SourceRange
}

type SourceRange struct {
	Filename   string
	Start, End SourcePos
}

type SourcePos struct {
	Line, Column, Byte int
}

// ValueSourceType describes what broad category of source location provided
// a particular value.
type ValueSourceType rune

const (
	// ValueFromUnknown is the zero value of ValueSourceType and is not valid.
	ValueFromUnknown ValueSourceType = 0

	// ValueFromConfig indicates that a value came from a .tf or .tf.json file,
	// e.g. the default value defined for a variable.
	ValueFromConfig ValueSourceType = 'C'

	// ValueFromAutoFile indicates that a value came from a "values file", like
	// a .tfvars file, that was implicitly loaded by naming convention.
	ValueFromAutoFile ValueSourceType = 'F'

	// ValueFromNamedFile indicates that a value came from a named "values file",
	// like a .tfvars file, that was passed explicitly on the command line (e.g.
	// -var-file=foo.tfvars).
	ValueFromNamedFile ValueSourceType = 'N'

	// ValueFromCLIArg indicates that the value was provided directly in
	// a CLI argument. The name of this argument is not recorded and so it must
	// be inferred from context.
	ValueFromCLIArg ValueSourceType = 'A'

	// ValueFromEnvVar indicates that the value was provided via an environment
	// variable. The name of the variable is not recorded and so it must be
	// inferred from context.
	ValueFromEnvVar ValueSourceType = 'E'

	// ValueFromInput indicates that the value was provided at an interactive
	// input prompt.
	ValueFromInput ValueSourceType = 'I'

	// ValueFromPlan indicates that the value was retrieved from a stored plan.
	ValueFromPlan ValueSourceType = 'P'

	// ValueFromCaller indicates that the value was explicitly overridden by
	// a caller to Context.SetVariable after the context was constructed.
	ValueFromCaller ValueSourceType = 'S'
)

// VarEnvPrefix is the prefix for environment variables that represent values
// for root module input variables.
const VarEnvPrefix = "TF_VAR_"

type UnparsedVariableValue interface {
	// ParseVariableValue information in the provided variable configuration
	// to parse (if necessary) and return the variable value encapsulated in
	// the receiver.
	//
	// If error diagnostics are returned, the resulting value may be invalid
	// or incomplete.
	ParseVariableValue(mode VariableParsingMode) (*InputValue, hcl.Diagnostics)
}

type unparsedVariableValueString struct {
	str        string
	name       string
	sourceType ValueSourceType
}

func (v unparsedVariableValueString) ParseVariableValue(mode VariableParsingMode) (*InputValue, hcl.Diagnostics) {

	val, hclDiags := mode.Parse(v.name, v.str)

	return &InputValue{
		Value:      val,
		SourceType: v.sourceType,
	}, hclDiags
}

// unparsedVariableValueLiteral is a backend.UnparsedVariableValue
// implementation that was actually already parsed (!). This is
// intended to deal with expressions inside "tfvars" files.
type unparsedVariableValueExpression struct {
	expr       hcl.Expression
	sourceType ValueSourceType
}

func (v unparsedVariableValueExpression) ParseVariableValue(mode VariableParsingMode) (*InputValue, hcl.Diagnostics) {
	val, hclDiags := v.expr.Value(nil) // nil because no function calls or variable references are allowed here

	rng := SourceRangeFromHCL(v.expr.Range())

	return &InputValue{
		Value:       val,
		SourceType:  v.sourceType,
		SourceRange: rng,
	}, hclDiags
}

func SourceRangeFromHCL(hclRange hcl.Range) SourceRange {
	return SourceRange{
		Filename: hclRange.Filename,
		Start: SourcePos{
			Line:   hclRange.Start.Line,
			Column: hclRange.Start.Column,
			Byte:   hclRange.Start.Byte,
		},
		End: SourcePos{
			Line:   hclRange.End.Line,
			Column: hclRange.End.Column,
			Byte:   hclRange.End.Byte,
		},
	}
}

type rawFlag struct {
	Name  string
	Value string
}

func (i *Interpreter) ParseVariables(env []string, rawFlags []rawFlag) (map[string]*InputValue, hcl.Diagnostics) {
	ret := map[string]*InputValue{}

	variables, diags := i.ProcessVariables(env, rawFlags)
	//TODO process diagnostics

	for s, value := range variables {
		inputValue, _ := value.(UnparsedVariableValue).ParseVariableValue(VariableParseHCL)
		//TODO process diagnostics

		ret[s] = inputValue
	}

	return ret, diags
}

//TODO rawflags should keep the order, as it should control variable overriding behavior
func (i *Interpreter) ProcessVariables(env []string, rawFlags []rawFlag) (map[string]UnparsedVariableValue, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	ret := map[string]UnparsedVariableValue{}

	// First we'll deal with environment variables, since they have the lowest
	// precedence.
	{
		for _, raw := range env {
			if !strings.HasPrefix(raw, VarEnvPrefix) {
				continue
			}
			raw = raw[len(VarEnvPrefix):] // trim the prefix

			eq := strings.Index(raw, "=")
			if eq == -1 {
				// Seems invalid, so we'll ignore it.
				continue
			}

			name := raw[:eq]
			rawVal := raw[eq+1:]

			ret[name] = unparsedVariableValueString{
				str:        rawVal,
				name:       name,
				sourceType: ValueFromEnvVar,
			}
		}
	}
	const DefaultVarsFilenameJSON = DefaultVarsFilename + ".json"

	// Next up we have some implicit files that are loaded automatically
	// if they are present. There's the original terraform.tfvars
	// (DefaultVarsFilename) along with the later-added search for all files
	// ending in .auto.tfvars.

	/* Terraform loads variables in the following order, with later sources taking precedence over earlier ones:

	Environment variables
	The terraform.tfvars file, if present.
	The terraform.tfvars.json file, if present.
	Any *.auto.tfvars or *.auto.tfvars.json files, processed in lexical order of their filenames.
	Any -var and -var-file options on the command line, in the order they are provided. (This includes variables set by a Terraform Cloud workspace.)
	*/

	for _, terraformFile := range i.TerraformModule.Files {
		if terraformFile.filename == DefaultVarsFilename {
			moreDiags := parseVars(terraformFile, ValueFromAutoFile, ret)
			diags = append(diags, moreDiags...)
		}
	}

	for _, terraformFile := range i.TerraformModule.Files {
		if terraformFile.filename == DefaultVarsFilenameJSON {
			moreDiags := parseVars(terraformFile, ValueFromAutoFile, ret)
			diags = append(diags, moreDiags...)
		}
	}

	//TODO iterate through auto vars files in lexical order
	for _, terraformFile := range i.TerraformModule.Files {
		if isAutoVarFile(terraformFile.filename) {
			moreDiags := parseVars(terraformFile, ValueFromAutoFile, ret)
			diags = append(diags, moreDiags...)
		}
	}

	// Finally we process values given explicitly on the command line, either
	// as individual literal settings or as additional files to read.
	for _, rawFlag := range rawFlags {
		switch rawFlag.Name {
		case "-var":
			// Value should be in the form "name=value", where value is a
			// raw string whose interpretation will depend on the variable's
			// parsing mode.
			raw := rawFlag.Value
			eq := strings.Index(raw, "=")
			if eq == -1 {
				log.Fatalf("%s,\n %s", "Invalid -var option",
					"The given -var option %q is not correctly specified. Must be a variable name and value separated by an equals sign, like -var=\"key=value\".")
			}
			name := raw[:eq]
			rawVal := raw[eq+1:]
			ret[name] = unparsedVariableValueString{
				str:        rawVal,
				name:       name,
				sourceType: ValueFromCLIArg,
			}

		case "-var-file":
			for _, terraformFile := range i.TerraformModule.Files {
				if terraformFile.filename == rawFlag.Value {
					moreDiags := parseVars(terraformFile, ValueFromNamedFile, ret)
					diags = append(diags, moreDiags...)
				}
			}

		default:
			// Should never happen; always a bug in the code that built up
			// the contents of m.variableArgs.
			log.Fatalf("unsupported variable option name %q (this is a bug in Terraform)", rawFlag.Name)
		}
	}

	return ret, diags
}

func parseVars(terraformFile *TerraformFile, sourceType ValueSourceType, to map[string]UnparsedVariableValue) hcl.Diagnostics {
	attrs, hclDiags := terraformFile.File.Body.JustAttributes()

	for name, attr := range attrs {
		to[name] = unparsedVariableValueExpression{
			expr:       attr.Expr,
			sourceType: sourceType,
		}
	}
	return hclDiags
}

// isAutoVarFile determines if the file ends with .auto.tfvars or .auto.tfvars.json
func isAutoVarFile(path string) bool {
	return strings.HasSuffix(path, AUTO_TF_VARS) ||
		strings.HasSuffix(path, AUTO_TF_VARS_JSON)
}

// VariableParsingMode defines how values of a particular variable given by
// text-only mechanisms (command line arguments and environment variables)
// should be parsed to produce the final value.
type VariableParsingMode rune

// VariableParseLiteral is a variable parsing mode that just takes the given
// string directly as a cty.String value.
const VariableParseLiteral VariableParsingMode = 'L'

// VariableParseHCL is a variable parsing mode that attempts to parse the given
// string as an HCL expression and returns the result.
const VariableParseHCL VariableParsingMode = 'H'

func (m VariableParsingMode) Parse(name, value string) (cty.Value, hcl.Diagnostics) {
	switch m {
	case VariableParseLiteral:
		return cty.StringVal(value), nil
	case VariableParseHCL:
		fakeFilename := fmt.Sprintf("<value for var.%s>", name)
		expr, diags := hclsyntax.ParseExpression([]byte(value), fakeFilename, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return cty.DynamicVal, diags
		}
		val, valDiags := expr.Value(nil)
		//val, valDiags := expr.Value(&evalContext)
		diags = append(diags, valDiags...)
		return val, diags
	default:
		// Should never happen
		panic(fmt.Errorf("Parse called on invalid VariableParsingMode %#v", m))
	}
}
