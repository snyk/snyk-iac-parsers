package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTerraformTfvarsFile(t *testing.T) {
	assert.True(t, isTerraformTfvarsFile("terraform.tfvars"))
	assert.True(t, isTerraformTfvarsFile("path/to/terraform.tfvars"))
	assert.False(t, isTerraformTfvarsFile("test_terraform.tfvars"))
}

func TestIsValidVariableFile(t *testing.T) {
	assert.True(t, IsValidVariableFile("path/to/terraform.tfvars"))
	assert.False(t, IsValidVariableFile("path/to/terraform.tfvars.json"))
	assert.True(t, IsValidVariableFile("path/to/test.tf"))
	assert.True(t, IsValidVariableFile("path/to/test.auto.tfvars"))
	assert.False(t, IsValidVariableFile("path/to/test.auto.tfvars.json"))
}

func TestIsValidTerraformFile(t *testing.T) {
	assert.False(t, IsValidTerraformFile("path/to/terraform.tfvars"))
	assert.False(t, IsValidTerraformFile("path/to/terraform.tfvars.json"))
	assert.True(t, IsValidTerraformFile("path/to/test.tf"))
	assert.False(t, IsValidTerraformFile("path/to/test.auto.tfvars"))
	assert.False(t, IsValidTerraformFile("path/to/test.auto.tfvars.json"))
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
	actual := OrderFilesByPriority(input)
	assert.Equal(t, expected, actual)
}
