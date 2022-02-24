package terraform

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestIsTerraformTfvarsFile(t *testing.T) {
	assert.True(t, isTerraformTfvarsFile("terraform.tfvars"))
	assert.True(t, isTerraformTfvarsFile(fmt.Sprintf("path%cto%cterraform.tfvars", os.PathSeparator, os.PathSeparator)))
	assert.False(t, isTerraformTfvarsFile("test_terraform.tfvars"))
	assert.True(t, isTerraformTfvarsFile("C:\\\\path\\\\to\\\\terraform.tfvars"))
}

func TestIsValidVariableFile(t *testing.T) {
	assert.True(t, isValidVariableFile(fmt.Sprintf("path%cto%cterraform.tfvars", os.PathSeparator, os.PathSeparator)))
	assert.False(t, isValidVariableFile(fmt.Sprintf("path%cto%cterraform.tfvars.json", os.PathSeparator, os.PathSeparator)))
	assert.True(t, isValidVariableFile(fmt.Sprintf("path%cto%ctest.tf", os.PathSeparator, os.PathSeparator)))
	assert.True(t, isValidVariableFile(fmt.Sprintf("path%cto%ctest.auto.tfvars", os.PathSeparator, os.PathSeparator)))
	assert.False(t, isValidVariableFile(fmt.Sprintf("path%cto%ctest.auto.tfvars.json", os.PathSeparator, os.PathSeparator)))
}

func TestIsValidTerraformFile(t *testing.T) {
	assert.False(t, isValidTerraformFile(fmt.Sprintf("path%cto%cterraform.tfvars", os.PathSeparator, os.PathSeparator)))
	assert.False(t, isValidTerraformFile(fmt.Sprintf("path%cto%cterraform.tfvars.json", os.PathSeparator, os.PathSeparator)))
	assert.True(t, isValidTerraformFile(fmt.Sprintf("path%cto%ctest.tf", os.PathSeparator, os.PathSeparator)))
	assert.False(t, isValidTerraformFile(fmt.Sprintf("path%cto%ctest.auto.tfvars", os.PathSeparator, os.PathSeparator)))
	assert.False(t, isValidTerraformFile(fmt.Sprintf("path%cto%ctest.auto.tfvars.json", os.PathSeparator, os.PathSeparator)))
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
