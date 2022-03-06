package terraform

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

var tfFileLocalSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "locals",
		},
	},
}
