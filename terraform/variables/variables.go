package variables

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	tfFiles "github.com/snyk/snyk-iac-parsers/terraform/files"
	"github.com/zclconf/go-cty/cty"
)

type VariableMap map[string]cty.Value

func ExtractFromFile(fileName string, file *hcl.File) (VariableMap, hcl.Diagnostics) {
	varMap := VariableMap{}

	var variables VariableMap
	var hclDiags hcl.Diagnostics
	if strings.HasSuffix(fileName, tfFiles.TF) {
		variables, hclDiags = extractFromTfFile(file)
		if hclDiags.HasErrors() {
			return varMap, hclDiags
		}
	} else if strings.HasSuffix(fileName, tfFiles.TFVARS) {
		variables, hclDiags = extractFromTfvarsFile(file)
	}

	varMap["var"] = cty.ObjectVal(variables)

	// TODO: remove temporary dummy locals and implement local values
	varMap["local"] = cty.ObjectVal(VariableMap{
		"dummy": cty.StringVal("dummy"),
	})

	return varMap, hclDiags
}

func MergeVariables(variableMaps map[string]VariableMap) VariableMap {
	combinedVariableMaps := make(VariableMap)

	combinedVars := make(map[string]cty.Value)

	fileNames := make([]string, 0, len(variableMaps))
	for fileName := range variableMaps {
		fileNames = append(fileNames, fileName)
	}

	prioritisedFileNames := tfFiles.OrderFilesByPriority(fileNames)

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
