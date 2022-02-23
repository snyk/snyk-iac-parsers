package variables

import "github.com/hashicorp/hcl/v2"

//Taken from https://github.com/hashicorp/terraform/blob/f266d1ee82d1fa4d882c146cc131fec4bef753cf/internal/configs/parser_config.go#L214
// tfFileSchema is the schema for the top-level of a config file. We use
// the low-level HCL API for this level so we can easily deal with each
// block type separately with its own decoding logic.
var tfFileVariableSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "variable",
			LabelNames: []string{"name"},
		},
	},
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
