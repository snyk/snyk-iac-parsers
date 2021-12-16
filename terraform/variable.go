package terraform

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

//Taken from https://github.com/hashicorp/terraform/blob/b91d9435ea9d4e8f1708a7092cd7d20d26f15e03/internal/configs/named_values.go#L20
// Variable represents a "variable" block in a module or file.
type Variable struct {
	Name        string
	Description string
	Default     cty.Value

	// Type is the concrete type of the variable value.
	Type cty.Type
	// ConstraintType is used for decoding and type conversions, and may
	// contain nested ObjectWithOptionalAttr types.
	ConstraintType cty.Type

	ParsingMode VariableParsingMode
	//Validations []*VariableValidation
	Sensitive   bool

	DescriptionSet bool
	SensitiveSet   bool

	// Nullable indicates that null is a valid value for this variable. Setting
	// Nullable to false means that the module can expect this variable to
	// never be null.
	Nullable    bool
	NullableSet bool

	DeclRange hcl.Range
	Block hcl.Block
}

func decodeVariableBlock(block *hcl.Block, override bool) (*Variable, hcl.Diagnostics) {
	v := &Variable{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}

	// Unless we're building an override, we'll set some defaults
	// which we might override with attributes below. We leave these
	// as zero-value in the override case so we can recognize whether
	// or not they are set when we merge.
	if !override {
		v.Type = cty.DynamicPseudoType
		//v.ConstraintType = cty.DynamicPseudoType
		//v.ParsingMode = VariableParseLiteral
	}

	content, diags := block.Body.Content(variableBlockSchema)


	if attr, exists := content.Attributes["type"]; exists {
		ty, parseMode, tyDiags := decodeVariableType(attr.Expr)
		diags = append(diags, tyDiags...)
		v.ConstraintType = ty
		v.Type = ty.WithoutOptionalAttributesDeep()
		v.ParsingMode = parseMode
	}


	if attr, exists := content.Attributes["default"]; exists {
		val, valDiags := attr.Expr.Value(nil)
		diags = append(diags, valDiags...)

		// Convert the default to the expected type so we can catch invalid
		// defaults early and allow later code to assume validity.
		// Note that this depends on us having already processed any "type"
		// attribute above.
		// However, we can't do this if we're in an override file where
		// the type might not be set; we'll catch that during merge.
		if v.ConstraintType != cty.NilType {
			var err error
			val, err = convert.Convert(val, v.ConstraintType)
			if err != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid default value for variable",
					Detail:   fmt.Sprintf("This default value is not compatible with the variable's type constraint: %s.", err),
					Subject:  attr.Expr.Range().Ptr(),
				})
				val = cty.DynamicVal
			}
		}

		if !v.Nullable && val.IsNull() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid default value for variable",
				Detail:   "A null default value is not valid when nullable=false.",
				Subject:  attr.Expr.Range().Ptr(),
			})
		}

		v.Default = val
	}

	return v, diags
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

func decodeVariableType(expr hcl.Expression) (cty.Type, VariableParsingMode, hcl.Diagnostics) {
	if exprIsNativeQuotedString(expr) {
		// If a user provides the pre-0.12 form of variable type argument where
		// the string values "string", "list" and "map" are accepted, we
		// provide an error to point the user towards using the type system
		// correctly has a hint.
		// Only the native syntax ends up in this codepath; we handle the
		// JSON syntax (which is, of course, quoted within the type system)
		// in the normal codepath below.
		val, diags := expr.Value(nil)
		if diags.HasErrors() {
			return cty.DynamicPseudoType, VariableParseHCL, diags
		}
		str := val.AsString()
		switch str {
		case "string":
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid quoted type constraints",
				Detail:   "Terraform 0.11 and earlier required type constraints to be given in quotes, but that form is now deprecated and will be removed in a future version of Terraform. Remove the quotes around \"string\".",
				Subject:  expr.Range().Ptr(),
			})
			return cty.DynamicPseudoType, VariableParseLiteral, diags
		case "list":
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid quoted type constraints",
				Detail:   "Terraform 0.11 and earlier required type constraints to be given in quotes, but that form is now deprecated and will be removed in a future version of Terraform. Remove the quotes around \"list\" and write list(string) instead to explicitly indicate that the list elements are strings.",
				Subject:  expr.Range().Ptr(),
			})
			return cty.DynamicPseudoType, VariableParseHCL, diags
		case "map":
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid quoted type constraints",
				Detail:   "Terraform 0.11 and earlier required type constraints to be given in quotes, but that form is now deprecated and will be removed in a future version of Terraform. Remove the quotes around \"map\" and write map(string) instead to explicitly indicate that the map elements are strings.",
				Subject:  expr.Range().Ptr(),
			})
			return cty.DynamicPseudoType, VariableParseHCL, diags
		default:
			return cty.DynamicPseudoType, VariableParseHCL, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Invalid legacy variable type hint",
				Detail:   `To provide a full type expression, remove the surrounding quotes and give the type expression directly.`,
				Subject:  expr.Range().Ptr(),
			}}
		}
	}

	// First we'll deal with some shorthand forms that the HCL-level type
	// expression parser doesn't include. These both emulate pre-0.12 behavior
	// of allowing a list or map of any element type as long as all of the
	// elements are consistent. This is the same as list(any) or map(any).
	switch hcl.ExprAsKeyword(expr) {
	case "list":
		return cty.List(cty.DynamicPseudoType), VariableParseHCL, nil
	case "map":
		return cty.Map(cty.DynamicPseudoType), VariableParseHCL, nil
	}

	ty, diags := typeexpr.TypeConstraint(expr)
	if diags.HasErrors() {
		return cty.DynamicPseudoType, VariableParseHCL, diags
	}

	switch {
	case ty.IsPrimitiveType():
		// Primitive types use literal parsing.
		return ty, VariableParseLiteral, diags
	default:
		// Everything else uses HCL parsing
		return ty, VariableParseHCL, diags
	}
}

func exprIsNativeQuotedString(expr hcl.Expression) bool {
	_, ok := expr.(*hclsyntax.TemplateExpr)
	return ok
}
