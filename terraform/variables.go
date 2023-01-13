package terraform

import (
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type ValueMap map[string]cty.Value

type ExpressionMap map[string]hcl.Expression

type ModuleVariables struct {
	inputs ValueMap
	locals ValueMap
}

type InputVariablesByFile map[string]ValueMap

// ExtractVariables extracts the input variables and local values from the provided file
func ExtractVariables(file File) (ValueMap, ExpressionMap, error) {
	inputsMap := ValueMap{}
	localsMap := ExpressionMap{}
	var hclDiags hcl.Diagnostics

	if isValidInputVariablesFile(file.fileName) {
		var inputHclDiags hcl.Diagnostics
		inputsMap, hclDiags = extractInputVariablesFromFile(file)
		hclDiags = append(hclDiags, inputHclDiags...)
	}

	if isValidTerraformFile(file.fileName) {
		var localsHclDiags hcl.Diagnostics
		localsMap, localsHclDiags = extractLocalsFromFile(file)
		hclDiags = append(hclDiags, localsHclDiags...)
	}

	return inputsMap, localsMap, hclDiags
}

func extractInputVariablesFromFile(file File) (ValueMap, hcl.Diagnostics) {
	var inputVariables ValueMap
	var hclDiags hcl.Diagnostics
	if strings.HasSuffix(file.fileName, TF) {
		inputVariables, hclDiags = extractInputVariablesFromTfFile(file.hclFile)
	} else if strings.HasSuffix(file.fileName, TFVARS) {
		inputVariables, hclDiags = extractInputVariablesFromTfvarsFile(file.hclFile)
	}

	if hclDiags.HasErrors() {
		return inputVariables, hclDiags
	}

	return inputVariables, hclDiags
}

func extractLocalsFromFile(file File) (ExpressionMap, hcl.Diagnostics) {
	localExprsMap := ExpressionMap{}

	bodyContent, _, hclDiags := file.hclFile.Body.PartialContent(tfFileLocalSchema)
	if hclDiags.HasErrors() {
		return localExprsMap, hclDiags
	}

	for _, block := range bodyContent.Blocks {
		attrs, _ := block.Body.JustAttributes()
		for localName, attr := range attrs {
			localExprsMap[localName] = attr.Expr
		}
	}

	return localExprsMap, hclDiags
}

// Logic inspired from https://github.com/hashicorp/terraform/blob/f266d1ee82d1fa4d882c146cc131fec4bef753cf/internal/configs/named_values.go#L113
func extractInputVariablesFromTfFile(file *hcl.File) (ValueMap, hcl.Diagnostics) {
	inputVariablesMap := ValueMap{}

	bodyContent, _, hclDiags := file.Body.PartialContent(tfFileVariableSchema)
	if hclDiags.HasErrors() {
		return inputVariablesMap, hclDiags
	}

	for _, block := range bodyContent.Blocks {
		name := block.Labels[0]

		attrs, _ := block.Body.JustAttributes()
		defaultValue := attrs["default"]
		if defaultValue != nil {
			value, diags := defaultValue.Expr.Value(&hcl.EvalContext{Functions: terraformFunctions})
			if diags.HasErrors() || value.IsNull() {
				continue
			}

			inputVariablesMap[name] = value
		}
	}

	return inputVariablesMap, hclDiags
}

func extractInputVariablesFromTfvarsFile(file *hcl.File) (ValueMap, hcl.Diagnostics) {
	inputVariablesMap := ValueMap{}

	attrs, hclDiags := file.Body.JustAttributes()

	for name, attr := range attrs {
		value, diags := attr.Expr.Value(&hcl.EvalContext{Functions: terraformFunctions})
		if diags.HasErrors() {
			continue
		}
		inputVariablesMap[name] = value
	}
	return inputVariablesMap, hclDiags
}

func mergeInputVariables(inputVariablesByFile InputVariablesByFile) ValueMap {
	combinedInputVariables := make(ValueMap)

	fileNames := make([]string, 0, len(inputVariablesByFile))

	for fileName := range inputVariablesByFile {
		fileNames = append(fileNames, fileName)
	}

	// The order of iteration over maps is not deterministic. In order for this
	// function to return deterministic results, sort the slice of file names
	// first.

	sort.Strings(fileNames)

	prioritisedFileNames := orderFilesByPriority(fileNames)

	for _, fileName := range prioritisedFileNames {
		inputVariablesMap := inputVariablesByFile[fileName]
		for input, value := range inputVariablesMap {
			combinedInputVariables[input] = value
		}
	}

	return combinedInputVariables
}

var maxLocalsDerefIterations = 32

func dereferenceLocals(localExprsMap ExpressionMap, inputs ValueMap) ValueMap {
	currLocalVals := ValueMap{}
	nextLocalVals := ValueMap{}

	for i := 0; i < maxLocalsDerefIterations; i++ {
		for localName, localExpr := range localExprsMap {
			newLocalVal, hclDiags := localExpr.Value(&hcl.EvalContext{
				Variables: createValueMap(ModuleVariables{
					inputs: inputs,
					locals: currLocalVals,
				}),
				Functions: terraformFunctions,
			})

			// the local cannot be dereferenced so move onto the next one
			if !newLocalVal.IsKnown() || hclDiags.HasErrors() {
				continue
			}

			// a local has been dereferenced so store it in the
			nextLocalVals[localName] = newLocalVal
		}

		// stop if the local values haven't changed between dereferencing loops
		if reflect.DeepEqual(currLocalVals, nextLocalVals) {
			break
		}

		for localName, localVal := range nextLocalVals {
			currLocalVals[localName] = localVal
		}
	}

	return nextLocalVals
}

func createValueMap(variables ModuleVariables) ValueMap {
	return ValueMap{
		"var":   cty.ObjectVal(variables.inputs),
		"local": cty.ObjectVal(variables.locals),
	}
}
