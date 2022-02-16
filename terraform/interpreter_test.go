package terraform

import (
	"os"
	"testing"
)

func TestInterpreter_ProcessDirectory(t *testing.T) {
	args := make([]rawFlag, 0)
	interpreter := NewInterpreter()
	bytes, _ := interpreter.ModuleAsJson("../../goof-cloud-config-terraform-langfeatures-demo/modules", os.Environ(), args)
	println(string(bytes))
}
