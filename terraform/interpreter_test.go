package terraform

import (
	"github.com/hashicorp/hcl/v2/hclparse"
	"testing"
)



func TestInterpreter_ProcessDirectory(t *testing.T) {
	interpreter := Interpreter{parser: hclparse.NewParser()}
	interpreter.ProcessDirectory("../../goof-cloud-config-terraform-langfeatures-demo/variables")
}
