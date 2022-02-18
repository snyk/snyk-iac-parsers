package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"strings"
)

type VariableMap map[string]cty.Value

func extractFromFile(fileName string, file *hcl.File) (VariableMap, hcl.Diagnostics) {
	varMap := VariableMap{}

	var variables VariableMap
	var hclDiags hcl.Diagnostics
	if strings.HasSuffix(fileName, TF) {
		variables, hclDiags = extractFromTfFile(file)
		if hclDiags.HasErrors() {
			return varMap, hclDiags
		}
	} else if strings.HasSuffix(fileName, TFVARS) {
		variables, hclDiags = extractFromTfvarsFile(file)
	}

	varMap["var"] = cty.ObjectVal(variables)

	// TODO: remove temporary dummy locals and implement local values
	varMap["local"] = cty.ObjectVal(VariableMap{
		"dummy": cty.StringVal("dummy"),
	})

	return varMap, hclDiags
}

// Logic inspired from https://github.com/hashicorp/terraform/blob/f266d1ee82d1fa4d882c146cc131fec4bef753cf/internal/configs/named_values.go#L113
func extractFromTfFile(file *hcl.File) (VariableMap, hcl.Diagnostics) {
	varMap := VariableMap{}

	bodyContent, _, hclDiags := file.Body.PartialContent(tfFileVariableSchema)
	if hclDiags.HasErrors() {
		return varMap, hclDiags
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

			varMap[name] = value
		}
	}

	return varMap, hclDiags
}

func extractFromTfvarsFile(file *hcl.File) (VariableMap, hcl.Diagnostics) {
	varMap := VariableMap{}

	attrs, hclDiags := file.Body.JustAttributes()

	for name, attr := range attrs {
		value, diags := attr.Expr.Value(&hcl.EvalContext{Functions: terraformFunctions})
		if diags.HasErrors() {
			continue
		}
		varMap[name] = value
	}
	return varMap, hclDiags
}

func mergeVariables(variableMaps map[string]VariableMap) VariableMap {
	combinedVariableMaps := make(VariableMap)

	combinedVars := make(map[string]cty.Value)

	fileNames := make([]string, 0, len(variableMaps))
	for fileName := range variableMaps {
		fileNames = append(fileNames, fileName)
	}

	prioritisedFileNames := orderFilesByPriority(fileNames)

	for _, fileName := range prioritisedFileNames {
		variableMap := variableMaps[fileName]["var"].AsValueMap()
		for variable, value := range variableMap {
			combinedVars[variable] = value
		}
	}

	combinedVariableMaps["var"] = cty.ObjectVal(combinedVars)

	// TODO: merge locals too once supported

	return combinedVariableMaps
}
