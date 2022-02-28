package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestParseHCL2JSONSuccess(t *testing.T) {
	type test struct {
		name      string
		input     string
		variables ModuleVariables
		expected  string
	}

	tests := []test{
		{
			name: "Nested block inside a labelled block",
			input: `
block "label_one" "label_two" {
	nested_block { }
}`,
			expected: `{
	"block": {
		"label_one": {
			"label_two": {
				"nested_block": {}
			}
		}
	}
}`,
		},
		{
			name: "Two simple blocks",
			input: `
block "label_one" {
}
block "label_one" {
}
`,
			expected: `{
	"block": {
		"label_one": [
			{},
			{}
		]
	}
}`,
		},
		{
			name: "Block with multiple labels and no attributes",
			input: `
resource "test1" "test2" {
}

resource "test1" "test3" {
}
`,
			expected: `{
	"resource": {
		"test1": {
			"test2": {},
			"test3": {}
		}
	}
}`,
		},
		{
			name: "Block with a literal value attribute",
			input: `
block "label_one" {
	attribute = "value"
}`,
			expected: `{
	"block": {
		"label_one": {
			"attribute": "value"
		}
	}
}`,
		},
		{
			name: "Block with a unary operation expression attribute",
			input: `
block "label_one" {
	attribute = -1
}`,
			expected: `{
	"block": {
		"label_one": {
			"attribute": -1
		}
	}
}`,
		},
		{
			name: "Block with a template expression attribute",
			input: `
block "label_one" {
	attribute = "${1 + 2}"
}`,
			expected: `{
	"block": {
		"label_one": {
			"attribute": 3
		}
	}
}`,
		},
		{
			name: "Block with a template expression attribute with a for loop wrapped in a string",
			input: `
block "label_one" {
	attribute = "This has a for loop: %{for x in [1, 2, 3]}${x},%{endfor}"
}`,
			expected: `{
	"block": {
		"label_one": {
			"attribute": "This has a for loop: 1,2,3,"
		}
	}
}`,
		},
		{
			name: "Block with a template expression attribute with an if statement wrapped in a string",
			input: `
block "label_one" {
	attribute = "This has an if statement: %{ if true }true%{ else }false%{ endif }"
}`,
			expected: `{
	"block": {
		"label_one": {
			"attribute": "This has an if statement: true"
		}
	}
}`,
		},
		{
			name: "Block with a conditional expression",
			input: `
block "label_one" {
	attribute = true ? "true" : "false"
}`,
			expected: `{
	"block": {
		"label_one": {
			"attribute": "true"
		}
	}
}`,
		},
		{
			name: "Block with a for expression attribute",
			input: `locals {
	thing = [for x in [1, 2, 3]: x * 2]
}`,
			expected: `{
	"locals": {
		"thing": [
			2,
			4,
			6
		]
	}
}`,
		},
		{
			name: "Block with a splat expression",
			input: `locals {
	thing = [{"id": 1}, {"id": 2}, {"id": 3}][*].id
}`,
			expected: `{
	"locals": {
		"thing": [
			1,
			2,
			3
		]
	}
}`,
		},
		{
			name: "Block with a scope traversal expression",
			input: `locals {
	thing = [1, 2, 3][1]
}`,
			expected: `{
	"locals": {
		"thing": 2
	}
}`,
		},
		{
			name: "Block with an object cons expression",
			input: `locals {
	thing = {
		a = "1"
		b = "2"
	}
}`,
			expected: `{
	"locals": {
		"thing": {
			"a": "1",
			"b": "2"
		}
	}
}`,
		},
		{
			name: "Block with heredoc",
			input: `
locals {
	heredoc = <<EOF
		Another heredoc, that
		doesn't remove indentation
	EOF
}`,
			expected: `{
	"locals": {
		"heredoc": "\t\tAnother heredoc, that\n\t\tdoesn't remove indentation\n"
	}
}`,
		},
		{
			name: "Block with a missing reference",
			input: `
locals {
	cond = test3 > 2 ? 1: 0
}`,
			expected: `{
	"locals": {
		"cond": "${test3 \u003e 2 ? 1: 0}"
	}
}`,
		},
		{
			name: "Block with functions to simplify",
			input: `locals {
		a = split("-", "xyx-abc-def")
		x = 1 + 2
		y = pow(2,3)
		t = "x=${4+abs(2-3)*parseint("02",16)}"
		j = jsonencode({
			a = "a"
			b = 5
		})
		with_vars = x + 1
	}`,
			expected: `{
	"locals": {
		"a": [
			"xyx",
			"abc",
			"def"
		],
		"j": "{\"a\":\"a\",\"b\":5}",
		"t": "x=6",
		"with_vars": "${x + 1}",
		"x": 3,
		"y": 8
	}
}`,
		},
		{
			name: "Block with nested attributes",
			input: `
locals {
	other = {
		3 = 1
		"a.b.c[\"hi\"][3].*" = 3
		a.b.c = "True"
	}
}`,
			expected: `{
	"locals": {
		"other": {
			"3": 1,
			"a.b.c": "True",
			"a.b.c[\"hi\"][3].*": 3
		}
	}
}`,
		},
		{
			name: "Local block referencing an attribute defined on itself",
			input: `
locals {
	x = -10
	y = -x
	z = -(1 + 4)
}`,
			expected: `{
	"locals": {
		"x": -10,
		"y": "${-x}",
		"z": -5
	}
}`,
		},
		{
			name: "Mixed types of blocks (data, variable) with referenced variables",
			input: `
data "terraform_remote_state" "remote" {
	backend = "s3"
	config = {
		profile = var.profile
		region  = var.region
		bucket  = "mybucket"
		key     = "mykey"
	}
}
variable "profile" {}
variable "region" {
	default = "us-east-1"
}`,
			expected: `{
	"data": {
		"terraform_remote_state": {
			"remote": {
				"backend": "s3",
				"config": {
					"bucket": "mybucket",
					"key": "mykey",
					"profile": "${var.profile}",
					"region": "${var.region}"
				}
			}
		}
	},
	"variable": {
		"profile": {},
		"region": {
			"default": "us-east-1"
		}
	}
}`,
		},
		{
			name: "Block referencing a defined input variable",
			input: `
 resource "aws_instance" "app_server" {
   ami           = "ami-08d70e59c07c61a3a"
   instance_type = "t2.micro"

   tags = {
    Name = var.instance_name
   }
 }`,
			expected: `{
	"resource": {
		"aws_instance": {
			"app_server": {
				"ami": "ami-08d70e59c07c61a3a",
				"instance_type": "t2.micro",
				"tags": {
					"Name": "test"
				}
			}
		}
	}
}`,
			variables: ModuleVariables{
				inputs: ValueMap{
					"instance_name": cty.StringVal("test"),
				},
			},
		},
		{
			name: "Block referencing a defined local variable",
			input: `
 resource "aws_instance" "app_server" {
   ami           = "ami-08d70e59c07c61a3a"
   instance_type = "t2.micro"

   tags = {
    Name = local.instance_name
   }
 }`,
			expected: `{
	"resource": {
		"aws_instance": {
			"app_server": {
				"ami": "ami-08d70e59c07c61a3a",
				"instance_type": "t2.micro",
				"tags": {
					"Name": "test"
				}
			}
		}
	}
}`,
			variables: ModuleVariables{
				locals: ValueMap{
					"instance_name": cty.StringVal("test"),
				},
			},
		},
		{
			name: "Variable block referencing a defined local variable",
			input: `
 	variable "test1" {
		type = "string"
   		test2 = local.instance_name
 	}`,
			expected: `{
	"variable": {
		"test1": {
			"test2": "test",
			"type": "string"
		}
	}
}`,
			variables: ModuleVariables{
				locals: ValueMap{
					"instance_name": cty.StringVal("test"),
				},
			},
		},
		{
			name: "Local block referencing a defined local variable",
			input: `
 	locals {
   		test = local.instance_name
 	}`,
			expected: `{
	"locals": {
		"test": "test"
	}
}`,
			variables: ModuleVariables{
				locals: ValueMap{
					"instance_name": cty.StringVal("test"),
				},
			},
		},
		{
			name: "Local block referencing a variable in the attribute key",
			input: `
locals {
	other = {	
		"${local.test2}" = 4
		3 = 1
		"local.test1" = 89
	}
}`,
			expected: `{
	"locals": {
		"other": {
			"3": 1,
			"b": 4,
			"local.test1": 89
		}
	}
}`,
			variables: ModuleVariables{
				locals: ValueMap{
					"test1": cty.StringVal("a"),
					"test2": cty.StringVal("b"),
				},
			},
		},
		{
			name: "Block referencing an undefined variable",
			input: `
 resource "aws_instance" "app_server" {
   ami           = "ami-08d70e59c07c61a3a"
   instance_type = "t2.micro"

   tags = {
    Name = var.instance_name
   }
 }`,
			expected: `{
	"resource": {
		"aws_instance": {
			"app_server": {
				"ami": "ami-08d70e59c07c61a3a",
				"instance_type": "t2.micro",
				"tags": {
					"Name": "${var.instance_name}"
				}
			}
		}
	}
}`,
			variables: ModuleVariables{
				inputs: ValueMap{
					"wrong_name": cty.StringVal("test"),
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ParseHclToJson("test", tc.input, tc.variables)
			require.Nil(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestParseHCL2JSONFailure(t *testing.T) {
	type test struct {
		name     string
		input    string
		expected string
	}
	tests := []test{
		{
			name: "Invalid HCL",
			input: `
block "label_one" {
	attribute = "value"
`,
			expected: "Invalid HCL provided",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseHclToJson("test", tc.input, ModuleVariables{})
			require.NotNil(t, err)
			assert.Equal(t, tc.expected, err.Error())
			assert.True(t, isUserError(err))
		})
	}
}
