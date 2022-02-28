package terraform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	filesystemFiles, filesystemExpected, err := setupFilesystemTests()
	assert.Nil(t, err)

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
			name:     "Multiple valid .tf files with default variables, a terraform.tfvars file and multiple *.auto.tfvars files",
			files:    filesystemFiles,
			expected: filesystemExpected,
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
		{
			name: "Correctly dereferencing local values in a single file",
			files: map[string]interface{}{
				"test.tf": `
				resource "aws_security_group" "allow_ssh" {
					name        = "allow_ssh"
					description = "Allow SSH inbound from anywhere"
					cidr_blocks = local.dummy
				}
				
				locals {
					dummy = "Dummy Value"
				}`,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test.tf": `{
	"locals": {
		"dummy": "Dummy Value"
	},
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "Dummy Value",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	}
}`,
				},
				"debugLogs": map[string]interface{}{},
			},
		},
		{
			name: "Correctly dereferencing local values in a multiple files",
			files: map[string]interface{}{
				"test.tf": `
				locals {
					dummy = "Dummy Value"
				}`,
				"test2.tf": `
				resource "aws_security_group" "allow_ssh" {
					name        = "allow_ssh"
					description = "Allow SSH inbound from anywhere"
					cidr_blocks = local.dummy
				}`,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{"test.tf": `{
	"locals": {
		"dummy": "Dummy Value"
	}
}`, "test2.tf": `{
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "Dummy Value",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	}
}`},
				"debugLogs": map[string]interface{}{},
			},
		},
		{
			name: "Correctly evaluating local expressions in a single file",
			files: map[string]interface{}{
				"test.tf": `
				resource "aws_security_group" "allow_ssh" {
					name        = "allow_ssh"
					description = "Allow SSH inbound from anywhere"
					cidr_blocks = local.dummy
				}
				
				locals {
					dummy = max(1+1, 999)
				}`,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test.tf": `{
	"locals": {
		"dummy": 999
	},
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": 999,
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	}
}`,
				},
				"debugLogs": map[string]interface{}{},
			},
		},
		{
			name: "Correctly dereferencing local variable that references an input variable in a single file",
			files: map[string]interface{}{
				"test.tf": `
				resource "aws_security_group" "allow_ssh" {
					name        = "allow_ssh"
					description = "Allow SSH inbound from anywhere"
					cidr_blocks = local.dummy
				}
				
				locals {
					dummy = var.dummy
				}
				
				variable "dummy" {
					default = "Dummy Value"
					type   = "string"
				}`,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{"test.tf": `{
	"locals": {
		"dummy": "Dummy Value"
	},
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "Dummy Value",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	},
	"variable": {
		"dummy": {
			"default": "Dummy Value",
			"type": "string"
		}
	}
}`},
				"debugLogs": map[string]interface{}{},
			},
		},
		{
			name: "Correctly dereferencing local variable that references other local variables",
			files: map[string]interface{}{
				"test1.tf": `
resource "aws_security_group" "allow_ssh" {
	name        = "allow_ssh"
	description = "Allow SSH inbound from anywhere"
	cidr_blocks = local.d9
}

locals {
	d8 = local.d7 + 1
	d6 = local.d5 + 1
	d4 = local.d3 + 1
	d2 = local.d1 + 1
}

variable "dummy" {
	default = 1
	type   = number
}`,
				"test2.tf": `
locals {
	d9 = local.d8 + 1
	d7 = local.d6 + 1
	d5 = local.d4 + 1
	d3 = local.d2 + 1
	d1 = var.dummy
}`,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test1.tf": `{
	"locals": {
		"d2": 2,
		"d4": 4,
		"d6": 6,
		"d8": 8
	},
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": 9,
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	},
	"variable": {
		"dummy": {
			"default": 1,
			"type": "${number}"
		}
	}
}`,
					"test2.tf": `{
	"locals": {
		"d1": 1,
		"d3": 3,
		"d5": 5,
		"d7": 7,
		"d9": 9
	}
}`},
				"debugLogs": map[string]interface{}{},
			},
		},
		{
			name: "Correctly dereferencing local variable that references other local variables in a function",
			files: map[string]interface{}{
				"test1.tf": `
resource "aws_security_group" "allow_ssh" {
	name        = "allow_ssh"
	description = "Allow SSH inbound from anywhere"
	cidr_blocks = local.d3
}

locals {
	d3 = local.d2 > local.d1 ? (local.d2 - local.d1) : (local.d1 - local.d2)
	d2 = 2
	d1 = 1
}`},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test1.tf": `{
	"locals": {
		"d1": 1,
		"d2": 2,
		"d3": 1
	},
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": 1,
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	}
}`},
				"debugLogs": map[string]interface{}{},
			},
		},
		{
			name: "Correctly stopping to dereference local variable that references too many other local variables",
			files: map[string]interface{}{
				"test1.tf": `
resource "aws_security_group" "allow_ssh" {
	name        = "allow_ssh"
	description = "Allow SSH inbound from anywhere"
	cidr_blocks = local.d34
}

locals {
	d34 = local.d33 + 1
	d33 = local.d32 + 1
	d32 = local.d31 + 1
	d31 = local.d30 + 1
	d30 = local.d29 + 1
	d29 = local.d28 + 1
	d28 = local.d27 + 1
	d27 = local.d26 + 1
	d26 = local.d25 + 1
	d25 = local.d24 + 1
	d24 = local.d23 + 1
	d23 = local.d22 + 1
	d22 = local.d21 + 1
	d21 = local.d20 + 1
	d20 = local.d19 + 1
	d19 = local.d18 + 1
	d18 = local.d17 + 1
	d17 = local.d16 + 1
	d16 = local.d15 + 1
	d15 = local.d14 + 1
	d14 = local.d13 + 1
	d13 = local.d12 + 1
	d12 = local.d11 + 1
	d11 = local.d10 + 1
	d10 = local.d9 + 1
	d9 = local.d8 + 1
	d8 = local.d7 + 1
	d7 = local.d6 + 1
	d6 = local.d5 + 1
	d5 = local.d4 + 1
	d4 = local.d3 + 1
	d3 = local.d2 + 1
	d2 = local.d1 + 1
	d1 = 1
}`,
			},
			expected: map[string]interface{}{
				"failedFiles": map[string]interface{}{},
				"parsedFiles": map[string]interface{}{
					"test1.tf": `{
	"locals": {
		"d1": 1,
		"d10": 10,
		"d11": 11,
		"d12": 12,
		"d13": 13,
		"d14": 14,
		"d15": 15,
		"d16": 16,
		"d17": 17,
		"d18": 18,
		"d19": 19,
		"d2": 2,
		"d20": 20,
		"d21": 21,
		"d22": 22,
		"d23": 23,
		"d24": 24,
		"d25": 25,
		"d26": 26,
		"d27": 27,
		"d28": 28,
		"d29": 29,
		"d3": 3,
		"d30": 30,
		"d31": 31,
		"d32": 32,
		"d33": 33,
		"d34": "${local.d33 + 1}",
		"d4": 4,
		"d5": 5,
		"d6": 6,
		"d7": 7,
		"d8": 8,
		"d9": 9
	},
	"resource": {
		"aws_security_group": {
			"allow_ssh": {
				"cidr_blocks": "${local.d34}",
				"description": "Allow SSH inbound from anywhere",
				"name": "allow_ssh"
			}
		}
	}
}`,
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
				parseHclToJson = func(fileName string, fileContent string, variableMap ModuleVariables) (string, error) {
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
				extractVariables = func(file File) (ValueMap, ExpressionMap, error) {
					if file.fileName == "fail.tf" {
						return nil, nil, tc.extractErr
					}
					return oldExtractVariables(file)
				}
			}
			actual := ParseModule(tc.files)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func setupFilesystemTests() (map[string]interface{}, map[string]interface{}, error) {
	filesystemExpected := map[string]interface{}{
		"debugLogs":   map[string]interface{}{},
		"failedFiles": map[string]interface{}{},
		"parsedFiles": map[string]interface{}{},
	}

	filesystemExpected["parsedFiles"].(map[string]interface{})[fmt.Sprintf("fixtures%cvariables.tf", os.PathSeparator)] = `{
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
}`
	filesystemExpected["parsedFiles"].(map[string]interface{})[fmt.Sprintf("fixtures%ctest.tf", os.PathSeparator)] = `{
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
}`

	// sets up the files that mimic data sent directly from the filesystem
	filesystemFiles := map[string]interface{}{}
	err := filepath.Walk("./fixtures", func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			filesystemFiles[filePath] = string(fileContent)
		}
		return nil
	})
	if err != nil {
		return filesystemFiles, nil, err
	}

	return filesystemFiles, filesystemExpected, nil
}
