package terraform

import (
	"reflect"
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
	prevLocalVals := ValueMap{}

	for i := 0; i < maxLocalsDerefIterations; i++ {
		for localName, localExpr := range localExprsMap {
			newLocalVal, hclDiags := localExpr.Value(&hcl.EvalContext{
				Variables: createValueMap(ModuleVariables{
					inputs: inputs,
					locals: currLocalVals,
				}),
				Functions: terraformFunctions,
			})

			if !newLocalVal.IsKnown() || hclDiags.HasErrors() {
				continue
			}

			currLocalVals[localName] = newLocalVal
		}

		// stop if the local values haven't changed between dereferencing loops
		if reflect.DeepEqual(currLocalVals, prevLocalVals) {
			break
		}

		for localName, localVal := range currLocalVals {
			prevLocalVals[localName] = localVal
		}
	}

	return currLocalVals
}

func createValueMap(variables ModuleVariables) ValueMap {
	return ValueMap{
		"var":   cty.ObjectVal(variables.inputs),
		"local": cty.ObjectVal(variables.locals),
	}
}
