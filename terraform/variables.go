package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type VariableMap map[string]cty.Value

func extractFromFile(file *hcl.File) (VariableMap, hcl.Diagnostics) {
	// TODO: remove temporary dummy locals
	// implement "local" when we do local values

	varMap, hclDiags := extractFromTfFile(file)
	if hclDiags.HasErrors() {
		return VariableMap{}, hclDiags
	}

	return VariableMap{
		"var": cty.ObjectVal(varMap),
		"local": cty.ObjectVal(VariableMap{
			"dummy": cty.StringVal("dummy"),
		}),
	}, hclDiags
}

// Logic inspired from https://github.com/hashicorp/terraform/blob/f266d1ee82d1fa4d882c146cc131fec4bef753cf/internal/configs/named_values.go#L113
func extractFromTfFile(file *hcl.File) (VariableMap, hcl.Diagnostics) {
	varMap := VariableMap{}
	bodyContent, _, hclDiags := file.Body.PartialContent(tfFileVariableSchema)

	if hclDiags.HasErrors() {
		return VariableMap{}, hclDiags
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
