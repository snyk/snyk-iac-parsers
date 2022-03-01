package terraform

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type ValueMap map[string]cty.Value

type ModuleVariables struct {
	inputs ValueMap
	locals ValueMap
}

type ParserVariables map[string]ValueMap

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
