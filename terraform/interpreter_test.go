package terraform

import (
	"log"
	"os"
	"testing"
)

func TestInterpreter_ProcessDirectory(t *testing.T) {
	interpreter := NewInterpreter()
	interpreter.ProcessDirectory("../../goof-cloud-config-terraform-langfeatures-demo/variables")
	interpreter.BuildModule()
	variables, _ := interpreter.ParseVariables(os.Environ(), []rawFlag{})
	merged := interpreter.TerraformModule.MergeVariables(variables)
	bytes, err := Convert(interpreter.TerraformModule, Options{Simplify: true, ContextVars: merged})
	if err != nil {
		log.Fatal(err)
	}
	println(string(bytes))
}
