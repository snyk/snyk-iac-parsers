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
			"dummy": cty.StringVal("dummy"),
		}),
		"local": cty.ObjectVal(VariableMap{
			"dummy": cty.StringVal("dummy"),
		}),
	}, nil
}

type VariableMap map[string]cty.Value
