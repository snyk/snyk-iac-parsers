package terraform

import (
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
	"testing"
)

func TestMergeVariables(t *testing.T) {
	input := []VariableMap{
		VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var1": cty.StringVal("val1"),
			}),
		},
		VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var2": cty.StringVal("val2"),
				"var3": cty.StringVal("val3"),
			}),
		},
		VariableMap{
			"var": cty.ObjectVal(VariableMap{
				"var2": cty.StringVal("val2-duplicate"),
			}),
		},
		VariableMap{
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
	actual := mergeVariables(input)
	assert.Equal(t, expected, actual)
}
