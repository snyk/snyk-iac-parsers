package terraform

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
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
	type TestInput struct {
		fileName    string
		fileContent string
	}

	type test struct {
		name     string
		input    TestInput
		expected VariableMap
	}
	tests := []test{
		{
			name: "Simple variable block with no default",
			input: TestInput{
				fileName: "test.tf",
				fileContent: `
				variable "test" {
					type = "string"
				}`,
			},
			expected: VariableMap{
				"var": cty.ObjectVal(VariableMap{}),
				"local": cty.ObjectVal(VariableMap{
					"dummy": cty.StringVal("dummy"),
				}),
			},
		},
		{
			name: "Simple variable block with default",
			input: TestInput{
				fileName: "test.tf",
				fileContent: `
				variable "test" {
					type = "string"
					default = "test"
				}`,
			},
			expected: VariableMap{
				"var": cty.ObjectVal(VariableMap{
					"test": cty.StringVal("test"),
				}),
				"local": cty.ObjectVal(VariableMap{
					"dummy": cty.StringVal("dummy"),
				}),
			},
		},
		{
			name: "Variable with null value",
			input: TestInput{
				fileName: "test.tf",
				fileContent: `
				variable "test" {
					type = "string"
					default = null
				}`,
			},
			expected: VariableMap{
				"var": cty.ObjectVal(VariableMap{}),
				"local": cty.ObjectVal(VariableMap{
					"dummy": cty.StringVal("dummy"),
				}),
			},
		},
		{
			name: "Two variable one with null value and the other with valid value",
			input: TestInput{
				fileName: "test.tf",
				fileContent: `
				variable "nullTest" {
					type = "string"
					default = null
				}
				
				variable "test" {
					type = "string"
					default = "test"
				}`,
			},
			expected: VariableMap{
				"var": cty.ObjectVal(VariableMap{
					"test": cty.StringVal("test"),
				}),
				"local": cty.ObjectVal(VariableMap{
					"dummy": cty.StringVal("dummy"),
				}),
			},
		},
		{
			name: "Non-variable block",
			input: TestInput{
				fileName: "test.tf",
				fileContent: `
				provider "google" {
					project = "acme-app"
					default  = "us-central1"
				}`,
			},
			expected: VariableMap{
				"var": cty.ObjectVal(VariableMap{}),
				"local": cty.ObjectVal(VariableMap{
					"dummy": cty.StringVal("dummy"),
				}),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := extractVariables(tc.input.fileName, tc.input.fileContent)
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
}

variable "dummy" {
	type = "string"
	default = "dummy"
}`
	jsonOutput := `{
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "dummy",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	},
	"variable": {
		"dummy": {
			"default": "dummy",
			"type": "string"
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
			name: "Multiple valid .tf files with dummy variables and no error",
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
				"debugLogs": map[string]interface{}{},
			},
		}, {
			name: "A valid .tf and terraform.tfvars file with no overlapping variables and no error",
			files: map[string]interface{}{
				"test.tf": `
resource "aws_security_group" "allow_ssh" {
	name        = "allow_ssh"
	description = "Allow SSH inbound from anywhere"
	cidr_blocks = var.dummy
}`,
				"terraform.tfvars": `dummy = "dummy"`,
			},
			expected: map[string]interface{}{
				"debugLogs":   map[string]interface{}{},
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test.tf": `{
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "dummy",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	}
}`,
				},
			},
		}, {
			name: "A valid .tf and random .tfvars file with overlapping variables and no error",
			files: map[string]interface{}{
				"test.tf": `
resource "aws_security_group" "allow_ssh" {
	name        = "allow_ssh"
	description = "Allow SSH inbound from anywhere"
	cidr_blocks = var.dummy
}

variable "dummy" {
	type = "string"
	default = "dummy"
}`,
				"test_terraform.tfvars": `dummy = "dummy_override"`,
			},
			expected: map[string]interface{}{
				"debugLogs":   map[string]interface{}{},
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test.tf": `{
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "dummy",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	},
	"variable": {
		"dummy": {
			"default": "dummy",
			"type": "string"
		}
	}
}`,
				},
			},
		}, {
			name: "A valid .tf and terraform.tfvars file with overlapping variables and no error",
			files: map[string]interface{}{
				"test.tf": `
resource "aws_security_group" "allow_ssh" {
	name        = "allow_ssh"
	description = "Allow SSH inbound from anywhere"
	cidr_blocks = var.dummy
}

variable "dummy" {
	type = "string"
	default = "dummy"
}`,
				"terraform.tfvars": `dummy = "dummy_override"`,
			},
			expected: map[string]interface{}{
				"debugLogs":   map[string]interface{}{},
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test.tf": `{
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "dummy_override",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	},
	"variable": {
		"dummy": {
			"default": "dummy",
			"type": "string"
		}
	}
}`,
				},
			},
		}, {
			name: "A valid .tf, terraform.tfvars, and *.auto.tfvars file with overlapping variables and no error",
			files: map[string]interface{}{
				"test.tf": `
resource "aws_security_group" "allow_ssh" {
	name        = "allow_ssh"
	description = "Allow SSH inbound from anywhere"
	cidr_blocks = var.dummy
}

variable "dummy" {
	type = "string"
	default = "dummy"
}`,
				"terraform.tfvars":   `dummy = "dummy_override"`,
				"b_test.auto.tfvars": `dummy = "b_dummy_override"`,
				"a_test.auto.tfvars": `dummy = "a_dummy_override"`,
			},
			expected: map[string]interface{}{
				"debugLogs":   map[string]interface{}{},
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test.tf": `{
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "b_dummy_override",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	},
	"variable": {
		"dummy": {
			"default": "dummy",
			"type": "string"
		}
	}
}`,
				},
			},
		}, {
			name: "Multiple valid .tf files with default variables, a terraform.tfvars file and multiple *.auto.tfvars files",
			files: map[string]interface{}{
				"variables.tf": `variable "remote_user_addr" {
  type = list(string)
  default = ["0.0.0.0/0"]
}

variable "remote_user_addr_terraform_tfvars" {
  type = list(string)
  default = ["1.2.3.4/32"]
}

variable "remote_user_addr_a_auto_tfvars" {
  type = list(string)
  default = ["1.2.3.4/32"]
}

variable "remote_user_addr_b_auto_tfvars" {
  type = list(string)
  default = ["1.2.3.4/32"]
}`,
				"test.tf": `resource "aws_security_group" "allow_ssh" {
  name        = "allow_ssh"
  description = "Allow SSH inbound from anywhere"
  vpc_id      = "${aws_vpc.main.id}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.remote_user_addr
  }
}

resource "aws_security_group" "allow_ssh_terraform_tfvars" {
  name        = "allow_ssh"
  description = "Allow SSH inbound from anywhere"
  vpc_id      = "${aws_vpc.main.id}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.remote_user_addr_terraform_tfvars
  }
}

resource "aws_security_group" "allow_ssh_a_auto_tfvars" {
  name        = "allow_ssh"
  description = "Allow SSH inbound from anywhere"
  vpc_id      = "${aws_vpc.main.id}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.remote_user_addr_a_auto_tfvars
  }
}

resource "aws_security_group" "allow_ssh_b_auto_tfvars" {
  name        = "allow_ssh"
  description = "Allow SSH inbound from anywhere"
  vpc_id      = "${aws_vpc.main.id}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.remote_user_addr_b_auto_tfvars
  }
}`,
				"terraform.tfvars": `remote_user_addr_terraform_tfvars = ["0.0.0.0/0"]

remote_user_addr_a_auto_tfvars = ["1.2.3.4/32"]

remote_user_addr_b_auto_tfvars = ["1.2.3.4/32"]

`,
				"a.auto.tfvars": `remote_user_addr_a_auto_tfvars = ["0.0.0.0/0"]

remote_user_addr_b_auto_tfvars = ["1.2.3.4/32"]
`,
				"b.auto.tfvars": `remote_user_addr_b_auto_tfvars = ["0.0.0.0/0"]
`,
			},
			expected: map[string]interface{}{
				"debugLogs":   map[string]interface{}{},
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"variables.tf": `{
	"variable": {
		"remote_user_addr": {
			"default": [
				"0.0.0.0/0"
			],
			"type": "${list(string)}"
		},
		"remote_user_addr_a_auto_tfvars": {
			"default": [
				"1.2.3.4/32"
			],
			"type": "${list(string)}"
		},
		"remote_user_addr_b_auto_tfvars": {
			"default": [
				"1.2.3.4/32"
			],
			"type": "${list(string)}"
		},
		"remote_user_addr_terraform_tfvars": {
			"default": [
				"1.2.3.4/32"
			],
			"type": "${list(string)}"
		}
	}
}`,
					"test.tf": `{
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"description": "Allow SSH inbound from anywhere",
				"ingress": {
					"cidr_blocks": [
						"0.0.0.0/0"
					],
					"from_port": 22,
					"protocol": "tcp",
					"to_port": 22
				},
				"name": "allow_ssh",
				"vpc_id": "${aws_vpc.main.id}"
			},
			"allow_ssh_a_auto_tfvars": {
				"description": "Allow SSH inbound from anywhere",
				"ingress": {
					"cidr_blocks": [
						"0.0.0.0/0"
					],
					"from_port": 22,
					"protocol": "tcp",
					"to_port": 22
				},
				"name": "allow_ssh",
				"vpc_id": "${aws_vpc.main.id}"
			},
			"allow_ssh_b_auto_tfvars": {
				"description": "Allow SSH inbound from anywhere",
				"ingress": {
					"cidr_blocks": [
						"0.0.0.0/0"
					],
					"from_port": 22,
					"protocol": "tcp",
					"to_port": 22
				},
				"name": "allow_ssh",
				"vpc_id": "${aws_vpc.main.id}"
			},
			"allow_ssh_terraform_tfvars": {
				"description": "Allow SSH inbound from anywhere",
				"ingress": {
					"cidr_blocks": [
						"0.0.0.0/0"
					],
					"from_port": 22,
					"protocol": "tcp",
					"to_port": 22
				},
				"name": "allow_ssh",
				"vpc_id": "${aws_vpc.main.id}"
			}
		}
	}
}`}},
		},
		{
			name: "Multiple files and one file with a user error at extract time",
			files: map[string]interface{}{
				"fail.tf":  fileContent,
				"test2.tf": fileContent,
			},
			extractErr: &CustomError{
				message: "User error",
				errors: []error{
					errors.New("Test"),
				},
				userError: true,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{
					"fail.tf": "User error",
				},
				"parsedFiles": map[string]interface{}{
					"test2.tf": jsonOutput,
				},
				"debugLogs": map[string]interface{}{
					"fail.tf": "\nTest",
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
				message: "Internal error",
				errors: []error{
					errors.New("Test"),
				},
				userError: false,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"fail.tf":  jsonOutput, // it's intentional for files that fail with internal errors at extraction time to still try to parse as the internal error can be a flake
					"test2.tf": jsonOutput,
				},
				"debugLogs": map[string]interface{}{},
			},
		},
		{
			name: "Multiple files and one file with a user error at parse time",
			files: map[string]interface{}{
				"fail.tf":  fileContent,
				"test2.tf": fileContent,
			},
			parseErr: &CustomError{
				message: "User error",
				errors: []error{
					errors.New("Test"),
				},
				userError: true,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{
					"fail.tf": "User error",
				},
				"parsedFiles": map[string]interface{}{
					"test2.tf": jsonOutput,
				},
				"debugLogs": map[string]interface{}{
					"fail.tf": "\nTest",
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
				message: "Internal error",
				errors: []error{
					errors.New("Test"),
				},
				userError: false,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test2.tf": jsonOutput,
				},
				"debugLogs": map[string]interface{}{},
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
