package terraform

//func TestExtractVariablesSuccess(t *testing.T) {
//	type TestInput struct {
//		file File
//	}
//
//	type test struct {
//		name             string
//		input            TestInput
//		expectedValueMap ValueMap
//	}
//	tests := []test{
//		{
//			name: "Simple variable block with no default",
//			input: TestInput{
//				file: File{
//					fileName: "test.tf",
//					fileContent: `
//					variable "test" {
//						type = "string"
//					}`,
//				},
//			},
//			expectedValueMap: ValueMap{},
//		},
//		{
//			name: "Simple variable block with default",
//			input: TestInput{
//				file: File{
//					fileName: "test.tf",
//					fileContent: `
//					variable "test" {
//						type = "string"
//						default = "test"
//					}`,
//				},
//			},
//			expectedValueMap: ValueMap{
//				"test": cty.StringVal("test"),
//			},
//		},
//		{
//			name: "Variable with null value",
//			input: TestInput{
//				file: File{
//					fileName: "test.tf",
//					fileContent: `
//					variable "test" {
//						type = "string"
//						default = null
//					}`,
//				},
//			},
//			expectedValueMap: ValueMap{},
//		},
//		{
//			name: "Two variable one with null value and the other with valid value",
//			input: TestInput{
//				file: File{
//					fileName: "test.tf",
//					fileContent: `
//					variable "nullTest" {
//						type = "string"
//						default = null
//					}
//
//					variable "test" {
//						type = "string"
//						default = "test"
//					}`,
//				},
//			},
//			expectedValueMap: ValueMap{
//				"test": cty.StringVal("test"),
//			},
//		},
//		{
//			name: "Non-variable block",
//			input: TestInput{
//				file: File{
//					fileName: "test.tf",
//					fileContent: `
//					provider "google" {
//						project = "acme-app"
//						default  = "us-central1"
//					}`,
//				},
//			},
//			expectedValueMap: ValueMap{},
//		},
//	}
//
//	for _, tc := range tests {
//		t.Run(tc.name, func(t *testing.T) {
//			hclFile, _ := hclsyntax.ParseConfig([]byte(tc.input.file.fileContent), tc.input.file.fileName, hcl.Pos{Line: 1, Column: 1})
//			tc.input.file.hclFile = hclFile
//			actualValueMap, _, err := extractVariables(tc.input.file)
//			require.Nil(t, err)
//			assert.Equal(t, tc.expectedValueMap, actualValueMap)
//			// the expression map returns pointers so it's easier to test end-to-end
//		})
//	}
//}

//func TestMergeVariablesFromTerraformFiles(t *testing.T) {
//	input := map[string]ValueMap{
//		"test1.tf": ValueMap{
//			"var1": cty.StringVal("val1"),
//		},
//		"test2.tf": ValueMap{
//			"var2": cty.StringVal("val2"),
//			"var3": cty.StringVal("val3"),
//		},
//		"test3.tf": ValueMap{
//			"var2": cty.StringVal("val2-duplicate"),
//		},
//		"test4.tf": ValueMap{},
//	}
//	expected := ValueMap{
//		"var1": cty.StringVal("val1"),
//		"var2": cty.StringVal("val2-duplicate"),
//		"var3": cty.StringVal("val3"),
//	}
//	actual := mergeInputVariables(input)
//	assert.Equal(t, expected, actual)
//}
//
//func TestMergeVariablesOverridesWithTerraformTfvars(t *testing.T) {
//	input := InputVariablesByFile{
//		"test1.tf": ValueMap{
//			"var": cty.StringVal("val1"),
//		},
//		"terraform.tfvars": ValueMap{
//			"var": cty.StringVal("val2"),
//		},
//	}
//	expected := ValueMap{
//		"var": cty.StringVal("val2"),
//	}
//	actual := mergeInputVariables(input)
//	assert.Equal(t, expected, actual)
//}
//
//func TestMergeVariablesOverridesWithAnyAutoTfvars(t *testing.T) {
//	input := InputVariablesByFile{
//		"test1.tf": ValueMap{
//			"var": cty.StringVal("val1"),
//		},
//		"terraform.tfvars": ValueMap{
//			"var": cty.StringVal("val2"),
//		},
//		"test.auto.tfvars": ValueMap{
//			"var": cty.StringVal("val3"),
//		},
//	}
//	expected := ValueMap{
//		"var": cty.StringVal("val3"),
//	}
//	actual := mergeInputVariables(input)
//	assert.Equal(t, expected, actual)
//}
//
//func TestMergeVariablesOverridesWithLexicalOrderAutoTfvars(t *testing.T) {
//	input := InputVariablesByFile{
//		"test1.tf": ValueMap{
//			"var": cty.StringVal("val1"),
//		},
//		"terraform.tfvars": ValueMap{
//			"var": cty.StringVal("val2"),
//		},
//		"test.auto.tfvars": ValueMap{
//			"var": cty.StringVal("val3"),
//		},
//		"a_test.auto.tfvars": ValueMap{
//			"var": cty.StringVal("val4"),
//		},
//	}
//	expected := ValueMap{
//		"var": cty.StringVal("val3"),
//	}
//	actual := mergeInputVariables(input)
//	assert.Equal(t, expected, actual)
//}
