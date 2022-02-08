package terraform

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
	"testing"
)

func TestParseHCL2JSONSuccess(t *testing.T) {
	type test struct {
		name      string
		input     string
		variables VariableMap
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
			variables: VariableMap{
				"var": cty.ObjectVal(VariableMap{
					"instance_name": cty.StringVal("test"),
				}),
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
			variables: VariableMap{
				"local": cty.ObjectVal(VariableMap{
					"instance_name": cty.StringVal("test"),
				}),
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
			variables: VariableMap{
				"local": cty.ObjectVal(VariableMap{
					"instance_name": cty.StringVal("test"),
				}),
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
			variables: VariableMap{
				"local": cty.ObjectVal(VariableMap{
					"instance_name": cty.StringVal("test"),
				}),
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
			variables: VariableMap{
				"local": cty.ObjectVal(VariableMap{
					"test1": cty.StringVal("a"),
					"test2": cty.StringVal("b"),
				}),
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
			variables: VariableMap{
				"var": cty.ObjectVal(VariableMap{
					"wrong_name": cty.StringVal("test"),
				}),
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
			_, err := ParseHclToJson("test", tc.input, nil)
			require.NotNil(t, err)
			assert.Equal(t, tc.expected, err.Error())
			assert.True(t, isUserError(err))
		})
	}
}

func TestExtractVariablesSuccess(t *testing.T) {
	type test struct {
		name     string
		input    string
		expected VariableMap
	}
	tests := []test{
		{
			name: "Simple variable block with no default",
			input: `
variable "test" {
	type = "string"
}`,
			expected: VariableMap{
				"var": cty.ObjectVal(VariableMap{
					"dummy": cty.StringVal("dummy_value"),
				}),
				"local": cty.ObjectVal(VariableMap{
					"dummy": cty.StringVal("dummy_local"),
				}),
			},
		},
		{
			name: "Simple variable block with default",
			input: `
variable "test" {
	type = "string"
	default = "test"
}`,
			expected: VariableMap{
				"var": cty.ObjectVal(VariableMap{
					"dummy": cty.StringVal("dummy_value"),
				}),
				"local": cty.ObjectVal(VariableMap{
					"dummy": cty.StringVal("dummy_local"),
				}),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := extractVariables("test", tc.input)
			require.Nil(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestParseModuleSuccess(t *testing.T) {
	fileContent := `
resource "aws_security_group" "allow_ssh" {
	name        = "allow_ssh"
	description = "Allow SSH inbound from anywhere"
	cidr_blocks = var.dummy
}`
	jsonOutput := `{
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "dummy_value",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	}
}`
	type test struct {
		name       string
		files      map[string]interface{}
		extractErr *CustomError
		parseErr   *CustomError
		expected   map[string]interface{}
	}
	tests := []test{
		{
			name: "Multiple valid files with dummy variables and no error",
			files: map[string]interface{}{
				"test1.tf": fileContent,
				"test2.tf": fileContent,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test1.tf": jsonOutput,
					"test2.tf": jsonOutput,
				},
			},
		},
		{
			name: "Multiple files and one file with a user error at extract time",
			files: map[string]interface{}{
				"fail.tf":  fileContent,
				"test2.tf": fileContent,
			},
			extractErr: &CustomError{
				message:   "User error",
				errors:    []error{},
				userError: true,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{
					"fail.tf": "User error",
				},
				"parsedFiles": map[string]interface{}{
					"test2.tf": jsonOutput,
				},
			},
		},
		{
			name: "Multiple files and one file with an internal error at extract time",
			files: map[string]interface{}{
				"fail.tf":  fileContent,
				"test2.tf": fileContent,
			},
			extractErr: &CustomError{
				message:   "Internal error",
				errors:    []error{},
				userError: false,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"fail.tf":  jsonOutput, // it's intentional for files that fail with internal errors at extraction time to still try to parse as the internal error can be a flake
					"test2.tf": jsonOutput,
				},
			},
		},
		{
			name: "Multiple files and one file with a user error at parse time",
			files: map[string]interface{}{
				"fail.tf":  fileContent,
				"test2.tf": fileContent,
			},
			parseErr: &CustomError{
				message:   "User error",
				errors:    []error{},
				userError: true,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{
					"fail.tf": "User error",
				},
				"parsedFiles": map[string]interface{}{
					"test2.tf": jsonOutput,
				},
			},
		},
		{
			name: "Multiple files and one file with an internal error at parse time",
			files: map[string]interface{}{
				"fail.tf":  fileContent,
				"test2.tf": fileContent,
			},
			parseErr: &CustomError{
				message:   "Internal error",
				errors:    []error{},
				userError: false,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test2.tf": jsonOutput,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.parseErr != nil {
				oldParseHclToJson := parseHclToJson
				defer func() {
					parseHclToJson = oldParseHclToJson
				}()
				parseHclToJson = func(fileName string, fileContent string, variableMap VariableMap) (string, error) {
					if fileName == "fail.tf" {
						return "", tc.parseErr
					}
					return oldParseHclToJson(fileName, fileContent, variableMap)
				}
			}
			if tc.extractErr != nil {
				oldExtractVariables := extractVariables
				defer func() {
					extractVariables = oldExtractVariables
				}()
				extractVariables = func(fileName string, fileContent string) (VariableMap, error) {
					if fileName == "fail.tf" {
						return nil, tc.extractErr
					}
					return oldExtractVariables(fileName, fileContent)
				}
			}
			actual := ParseModule(tc.files)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
