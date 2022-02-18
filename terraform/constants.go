package terraform

const (
	TF          = ".tf"
	TFVARS      = ".tfvars"
	AUTO_TFVARS = ".auto.tfvars"

	DEFAULT_TFVARS = "terraform.tfvars"
)

var VALID_VARIABLE_FILES = [...]string{TF, AUTO_TFVARS}
var VALID_TERRAFORM_FILES = [...]string{TF}
