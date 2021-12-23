package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

// Variable represents a "variable" block in a module or file.
type Variable struct {
	Name       string
	Sensitive  bool
	Default    cty.Value
	DeclRange  hcl.Range
	DefaultSet bool
}

var variableBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "description",
		},
		{
			Name: "default",
		},
		{
			Name: "type",
		},
		{
			Name: "sensitive",
		},
		{
			Name: "nullable",
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "validation",
		},
	},
}

func decodeVariableBlock(block *hcl.Block, override bool) (Variable, hcl.Diagnostics) {
	v := Variable{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}

	content, diags := block.Body.Content(variableBlockSchema)

	if attr, exists := content.Attributes["sensitive"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &v.Sensitive)
		diags = append(diags, valDiags...)
	}

	if attr, exists := content.Attributes["default"]; exists {
		val, valDiags := attr.Expr.Value(nil)
		diags = append(diags, valDiags...)
		v.Default = val
		v.DefaultSet = true
	}

	return v, diags
}

func (m *TerraformModule) MergeVariables(inputs map[string]*InputValue) map[string]cty.Value {
	ret := make(variableMap)

	vars := make(variableMap)
	//Handle variable default values
	for _, variable := range m.variables {
		if variable.DefaultSet {
			vars[variable.Name] = variable.Default
		}
	}
	//Override variable defaults with input
	for name, inputValue := range inputs {
		vars[name] = inputValue.Value
	}
	ret["var"] = cty.ObjectVal(vars)

	locals := make(variableMap)

	//Handle locals
	for _, local := range m.locals {
		locals[local.Name] = local.Value
	}

	ret["local"] = cty.ObjectVal(locals)

	return ret
}
