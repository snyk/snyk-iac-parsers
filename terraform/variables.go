package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func extractFromFile(file *hcl.File) (VariableMap, hcl.Diagnostics) {
	// TODO: remove temporary dummy variables
	// implement "var" when we do input variables and default values
	// implement "local" when we do local values
	return VariableMap{
		"var": cty.ObjectVal(VariableMap{
			"dummy": cty.StringVal("dummy_value"),
		}),
		"local": cty.ObjectVal(VariableMap{
			"dummy": cty.StringVal("dummy_local"),
		}),
	}, nil
}

type VariableMap map[string]cty.Value

func mergeVariables(variableMaps []VariableMap) VariableMap {
	combinedVariableMaps := make(VariableMap)

	combinedVars := make(VariableMap)
	for _, variableMap := range variableMaps {
		vars := variableMap["var"].AsValueMap()
		for variable, value := range vars {
			combinedVars[variable] = value
		}
	}

	combinedVariableMaps["var"] = cty.ObjectVal(combinedVars)

	// TODO: merge locals too once supported

	return combinedVariableMaps
}
