package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestMergeVariablesFromTerraformFiles(t *testing.T) {
	input := map[string]ValueMap{
		"test1.tf": ValueMap{
			"var1": cty.StringVal("val1"),
		},
		"test2.tf": ValueMap{
			"var2": cty.StringVal("val2"),
			"var3": cty.StringVal("val3"),
		},
		"test3.tf": ValueMap{
			"var2": cty.StringVal("val2-duplicate"),
		},
		"test4.tf": ValueMap{},
	}
	expected := ValueMap{
		"var1": cty.StringVal("val1"),
		"var2": cty.StringVal("val2-duplicate"),
		"var3": cty.StringVal("val3"),
	}
	actual := mergeInputVariables(input)
	assert.Equal(t, expected, actual)
}

func TestMergeVariablesOverridesWithTerraformTfvars(t *testing.T) {
	input := InputVariablesByFile{
		"test1.tf": ValueMap{
			"var": cty.StringVal("val1"),
		},
		"terraform.tfvars": ValueMap{
			"var": cty.StringVal("val2"),
		},
	}
	expected := ValueMap{
		"var": cty.StringVal("val2"),
	}
	actual := mergeInputVariables(input)
	assert.Equal(t, expected, actual)
}

func TestMergeVariablesOverridesWithAnyAutoTfvars(t *testing.T) {
	input := InputVariablesByFile{
		"test1.tf": ValueMap{
			"var": cty.StringVal("val1"),
		},
		"terraform.tfvars": ValueMap{
			"var": cty.StringVal("val2"),
		},
		"test.auto.tfvars": ValueMap{
			"var": cty.StringVal("val3"),
		},
	}
	expected := ValueMap{
		"var": cty.StringVal("val3"),
	}
	actual := mergeInputVariables(input)
	assert.Equal(t, expected, actual)
}

func TestMergeVariablesOverridesWithLexicalOrderAutoTfvars(t *testing.T) {
	input := InputVariablesByFile{
		"test1.tf": ValueMap{
			"var": cty.StringVal("val1"),
		},
		"terraform.tfvars": ValueMap{
			"var": cty.StringVal("val2"),
		},
		"test.auto.tfvars": ValueMap{
			"var": cty.StringVal("val3"),
		},
		"a_test.auto.tfvars": ValueMap{
			"var": cty.StringVal("val4"),
		},
	}
	expected := ValueMap{
		"var": cty.StringVal("val3"),
	}
	actual := mergeInputVariables(input)
	assert.Equal(t, expected, actual)
}
