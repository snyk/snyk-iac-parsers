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

type InputsByFile map[string]ValueMap

func extractInputsFromFile(fileName string, file *hcl.File) (ValueMap, hcl.Diagnostics) {
	var inputs ValueMap
	var hclDiags hcl.Diagnostics
	if strings.HasSuffix(fileName, TF) {
		inputs, hclDiags = extractInputsFromTfFile(file)
	} else if strings.HasSuffix(fileName, TFVARS) {
		inputs, hclDiags = extractInputsFromTfvarsFile(file)
	}

	if hclDiags.HasErrors() {
		return inputs, hclDiags
	}

	return inputs, hclDiags
}

// Logic inspired from https://github.com/hashicorp/terraform/blob/f266d1ee82d1fa4d882c146cc131fec4bef753cf/internal/configs/named_values.go#L113
func extractInputsFromTfFile(file *hcl.File) (ValueMap, hcl.Diagnostics) {
	inputsMap := ValueMap{}

	bodyContent, _, hclDiags := file.Body.PartialContent(tfFileVariableSchema)
	if hclDiags.HasErrors() {
		return inputsMap, hclDiags
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

			inputsMap[name] = value
		}
	}

	return inputsMap, hclDiags
}

func extractInputsFromTfvarsFile(file *hcl.File) (ValueMap, hcl.Diagnostics) {
	inputsMap := ValueMap{}

	attrs, hclDiags := file.Body.JustAttributes()

	for name, attr := range attrs {
		value, diags := attr.Expr.Value(&hcl.EvalContext{Functions: terraformFunctions})
		if diags.HasErrors() {
			continue
		}
		inputsMap[name] = value
	}
	return inputsMap, hclDiags
}

func mergeInputs(inputsByFile InputsByFile) ValueMap {
	combinedInputs := make(ValueMap)

	fileNames := make([]string, 0, len(inputsByFile))
	for fileName := range inputsByFile {
		fileNames = append(fileNames, fileName)
	}

	prioritisedFileNames := orderFilesByPriority(fileNames)

	for _, fileName := range prioritisedFileNames {
		inputsMap := inputsByFile[fileName]
		for input, value := range inputsMap {
			combinedInputs[input] = value
		}
	}

	return combinedInputs
}
