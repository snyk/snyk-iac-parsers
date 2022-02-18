package terraform

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsTerraformTfvarsFile(t *testing.T) {
	assert.True(t, isTerraformTfvarsFile("terraform.tfvars"))
	assert.True(t, isTerraformTfvarsFile("path/to/terraform.tfvars"))
	assert.False(t, isTerraformTfvarsFile("test_terraform.tfvars"))
}

func TestIsValidVariableFile(t *testing.T) {
	assert.True(t, isValidVariableFile("path/to/terraform.tfvars"))
	assert.False(t, isValidVariableFile("path/to/terraform.tfvars.json"))
	assert.True(t, isValidVariableFile("path/to/test.tf"))
	assert.True(t, isValidVariableFile("path/to/test.auto.tfvars"))
	assert.False(t, isValidVariableFile("path/to/test.auto.tfvars.json"))
}

func TestIsValidTerraformFile(t *testing.T) {
	assert.False(t, isValidTerraformFile("path/to/terraform.tfvars"))
	assert.False(t, isValidTerraformFile("path/to/terraform.tfvars.json"))
	assert.True(t, isValidTerraformFile("path/to/test.tf"))
	assert.False(t, isValidTerraformFile("path/to/test.auto.tfvars"))
	assert.False(t, isValidTerraformFile("path/to/test.auto.tfvars.json"))
}

func TestOrderFilesByPriority(t *testing.T) {
	input := []string{
		"c.auto.tfvars",
		"b.tf",
		"a.tf",
		"terraform.tfvars",
		"d.tf",
		"random",
		"b.auto.tfvars",
		"c.tf",
		"a.auto.tfvars",
	}
	expected := []string{
		"random",
		"b.tf",
		"a.tf",
		"d.tf",
		"c.tf",
		"terraform.tfvars",
		"a.auto.tfvars",
		"b.auto.tfvars",
		"c.auto.tfvars",
	}
	actual := orderFilesByPriority(input)
	assert.Equal(t, expected, actual)
}
