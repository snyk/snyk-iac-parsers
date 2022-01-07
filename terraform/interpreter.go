package terraform

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Interpreter struct {
	parser          *hclparse.Parser
	TerraformModule *TerraformModule
}

type FileSuffix string

const (
	HCL2              = ".tf"
	JSON              = ".json"
	TF_VARS           = ".tfvars"
	TF_VARS_JSON      = ".tfvars.json"
	AUTO_TF_VARS      = ".auto.tfvars"
	AUTO_TF_VARS_JSON = ".auto.tfvars.json"
)

// DefaultVarsFilename is the default filename used for vars
const DefaultVarsFilename = "terraform.tfvars"

func NewInterpreter() Interpreter {
	interpreter := Interpreter{}
	interpreter.parser = hclparse.NewParser()
	interpreter.TerraformModule = &TerraformModule{}
	return interpreter
}

func (i *Interpreter) ProcessDirectory(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		// Skip non-top-level files
		if file.IsDir() || strings.ContainsAny(file.Name(), "/") {
			continue
		}
		if strings.HasSuffix(file.Name(), HCL2) || strings.HasSuffix(file.Name(), TF_VARS) {
			i.parseHCLFile(filepath.Join(dir, file.Name()))
		} else if strings.HasSuffix(file.Name(), JSON) {
			//i.ParseJSONFile(filepath.Join(dir, file.Name()))
		}

	}
}

func (i *Interpreter) BuildModule() *TerraformModule {
	for filename, hclFile := range i.parser.Files() {
		var bodyContent *hcl.BodyContent
		var diags hcl.Diagnostics
		if strings.HasSuffix(filename, TF_VARS) || strings.HasSuffix(filename, TF_VARS_JSON) {
			bodyContent, _, _ = hclFile.Body.PartialContent(&hcl.BodySchema{
				Blocks: []hcl.BlockHeaderSchema{
					{
						Type:       "variable",
						LabelNames: []string{"name"},
					},
				},
			})
			i.addFileToModule(filename, hclFile, bodyContent, false)
		} else {
			bodyContent, diags = hclFile.Body.Content(configFileSchema)
			//TODO There might be var files with none standard name
			handleDiagnostics("Validation issue", diags, filename)
			i.addFileToModule(filename, hclFile, bodyContent, true)
		}

	}
	return i.TerraformModule
}

func (i *Interpreter) addFileToModule(filename string, hclFile *hcl.File, bodyContent *hcl.BodyContent, isConfig bool) {
	if !isOverrideFile(filename) {
		i.TerraformModule.addFile(hclFile, bodyContent, filename, isConfig)
	} else {
		log.Fatal("File overrides not implemented yet!")
	}
}

func handleDiagnostics(issue string, diags hcl.Diagnostics, filename string) {
	if diags != nil {
		if diags.HasErrors() {
			log.Fatal(diags)
		} else {
			log.Printf("%s, file: %s, %s", issue, filename, diags)
		}
	}
}

func isOverrideFile(filename string) bool {
	//TODO implement!!!
	return false
}

func (i *Interpreter) parseHCLFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	// Just keep the file name, instead of the whole path
	_, filename = path.Split(filename)
	i.ParseHCL(data, filename)
}

func (i *Interpreter) ParseHCL(src []byte, filename string) {
	_, diags := i.parser.ParseHCL(src, filename)
	handleDiagnostics("Parsing issue", diags, filename)
}

func (i *Interpreter) ParseJSONFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	// Just keep the file name, instead of the whole path
	_, filename = path.Split(filename)
	i.ParseJSON(data, filename)
}

func (i *Interpreter) ParseJSON(src []byte, filename string) {
	log.Fatal("Not implemented")
}
/*
  const module1 = {
  files: {
    file1: "Content1",
    file2: "Content2"
  },
  flags:{
   name: "Value"
  },
  env: ["name=value"]
};

 */

type Config struct {
	files map[string]string `json:"files"`
	flags map[string]string `json:"flags"`
	env []string `json:"env"`
}

func ParseModule(config map[string]interface{}) string{
	interpreter := NewInterpreter()
	for k,v := range config["files"].(map[string]interface{}){

		if strings.HasSuffix(k, HCL2) || strings.HasSuffix(k, TF_VARS) {
			interpreter.ParseHCL([]byte(v.(string)), k)
		} else if strings.HasSuffix(k, JSON) {
			interpreter.ParseJSON([]byte(v.(string)), k)
		}
	}
	interpreter.BuildModule()

	variables, _ := interpreter.ParseVariables(convert2ArrayOfString(config["env"].([]interface{})), convert2MapOfString(config["flags"].(map[string]interface{})))
	merged := interpreter.TerraformModule.MergeVariables(variables)
	bytes, err := Convert(interpreter.TerraformModule, Options{Simplify: true, ContextVars: merged})
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes)
}


func convert2ArrayOfString(t []interface{})[]string{
	s := make([]string, len(t))
	for i, v := range t {
		s[i] = fmt.Sprint(v)
	}
	return s
}

func convert2MapOfString(t map[string]interface{}) map[string]string{
	s := make(map[string]string, len(t))
	for i, v := range t {
		s[i] = fmt.Sprint(v)
	}
	return s
}