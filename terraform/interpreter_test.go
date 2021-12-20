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
	interpreter.ReadVariables(os.Environ(),[]rawFlag{})
	bytes, err := Convert(interpreter.TerraformModule, Options{Simplify: true})
	if err != nil {
		log.Fatal(err)
	}
	println(string(bytes))
}
