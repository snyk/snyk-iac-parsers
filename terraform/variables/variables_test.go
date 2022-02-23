package variables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestMergeVariablesFromTerraformFiles(t *testing.T) {
	input := map[string]VariableMap{
		"test1.tf": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var1": cty.StringVal("val1"),
			}),
		},
		"test2.tf": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var2": cty.StringVal("val2"),
				"var3": cty.StringVal("val3"),
			}),
		},
		"test3.tf": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var2": cty.StringVal("val2-duplicate"),
			}),
		},
		"test4.tf": VariableMap{
			"var": cty.ObjectVal(VariableMap{}),
		},
	}
	expected := VariableMap{
		"var": cty.ObjectVal(VariableMap{
			"var1": cty.StringVal("val1"),
			"var2": cty.StringVal("val2-duplicate"),
			"var3": cty.StringVal("val3"),
		}),
	}
	actual := MergeVariables(input)
	assert.Equal(t, expected, actual)
}

func TestMergeVariablesOverridesWithTerraformTfvars(t *testing.T) {
	input := map[string]VariableMap{
		"test1.tf": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var": cty.StringVal("val1"),
			}),
		},
		"terraform.tfvars": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var": cty.StringVal("val2")}),
		},
	}
	expected := VariableMap{
		"var": cty.ObjectVal(VariableMap{
			"var": cty.StringVal("val2"),
		}),
	}
	actual := MergeVariables(input)
	assert.Equal(t, expected, actual)
}

func TestMergeVariablesOverridesWithAnyAutoTfvars(t *testing.T) {
	input := map[string]VariableMap{
		"test1.tf": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var": cty.StringVal("val1"),
			}),
		},
		"terraform.tfvars": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var": cty.StringVal("val2")}),
		},
		"test.auto.tfvars": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var": cty.StringVal("val3")}),
		},
	}
	expected := VariableMap{
		"var": cty.ObjectVal(VariableMap{
			"var": cty.StringVal("val3"),
		}),
	}
	actual := MergeVariables(input)
	assert.Equal(t, expected, actual)
}

func TestMergeVariablesOverridesWithLexicalOrderAutoTfvars(t *testing.T) {
	input := map[string]VariableMap{
		"test1.tf": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var": cty.StringVal("val1"),
			}),
		},
		"terraform.tfvars": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var": cty.StringVal("val2")}),
		},
		"test.auto.tfvars": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var": cty.StringVal("val3")}),
		},
		"a_test.auto.tfvars": VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var": cty.StringVal("val4")}),
		},
	}
	expected := VariableMap{
		"var": cty.ObjectVal(VariableMap{
			"var": cty.StringVal("val3"),
		}),
	}
	actual := MergeVariables(input)
	assert.Equal(t, expected, actual)
}
