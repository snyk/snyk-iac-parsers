package terraform

import (
	"github.com/hashicorp/hcl/v2/hclparse"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Interpreter struct {
	parser *hclparse.Parser
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

type TerraformModule struct {
	Files []TerraformFile
}

type TerraformFile struct {
	Variables []*Variable
	Locals    []*Local
	Outputs   []*Output
	Resources []*Resource
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
			i.ParseHCLFile(filepath.Join(dir, file.Name()))
		} else if strings.HasSuffix(file.Name(), JSON) {
			i.ParseJSONFile(filepath.Join(dir, file.Name()))
		}

	}
	i.BuildModule()
}

func (i *Interpreter) BuildModule() {
	for filename, hclFile := range i.parser.Files() {
		content, diags := hclFile.Body.Content(configFileSchema)
		if diags != nil {
			if diags.HasErrors(){
				log.Fatal(diags)
			}else{
				log.Printf("File: %s parsing issue: %s",filename, diags)
			}
		}
		override:= isOverrideFile(filename)
		file := TerraformFile{}

		for _, block := range content.Blocks {
			switch block.Type {

			case "variable":
				variable, cfgDiags := decodeVariableBlock(block, override)
				diags = append(diags, cfgDiags...)
				if variable != nil {
					file.Variables = append(file.Variables, variable)
				}

			case "locals":
				locals, defsDiags := decodeLocalsBlock(block)
				diags = append(diags, defsDiags...)
				file.Locals = append(file.Locals, locals...)

			case "output":
				output, cfgDiags := decodeOutputBlock(block, override)
				diags = append(diags, cfgDiags...)
				if output != nil {
					file.Outputs = append(file.Outputs, output)
				}

			case "resource":
				resource, cfgDiags := decodeResourceBlock(block)
				diags = append(diags, cfgDiags...)
				if resource != nil {
					file.Resources = append(file.Resources, resource)
				}

			default:
				// Should never happen because the above cases should be exhaustive
				// for all block type names in our schema.
				continue

			}
		}

	}
}

//func (i *Interpreter) BuildModule() {
//	for filename, hclFile := range i.parser.Files() {
//		content, diags := hclFile.Body.Content(configFileSchema)
//		if diags != nil {
//			if diags.HasErrors(){
//				log.Fatal(diags)
//			}else{
//				log.Printf("File: %s parsing issue: %s",filename, diags)
//			}
//		}
//		override:= isOverrideFile(filename)
//		file := TerraformFile{}
//
//		for _, block := range content.Blocks {
//			switch block.Type {
//
//			case "variable":
//				cfg, cfgDiags := decodeVariableBlock(block, override)
//				diags = append(diags, cfgDiags...)
//				if cfg != nil {
//					file.Variables = append(file.Variables, cfg)
//				}
//
//			case "locals":
//				defs, defsDiags := decodeLocalsBlock(block)
//				diags = append(diags, defsDiags...)
//				file.Locals = append(file.Locals, defs...)
//
//			case "output":
//				cfg, cfgDiags := decodeOutputBlock(block, override)
//				diags = append(diags, cfgDiags...)
//				if cfg != nil {
//					file.Outputs = append(file.Outputs, cfg)
//				}
//
//			case "module":
//				cfg, cfgDiags := decodeModuleBlock(block, override)
//				diags = append(diags, cfgDiags...)
//				if cfg != nil {
//					file.ModuleCalls = append(file.ModuleCalls, cfg)
//				}
//
//			case "resource":
//				cfg, cfgDiags := decodeResourceBlock(block)
//				diags = append(diags, cfgDiags...)
//				if cfg != nil {
//					file.ManagedResources = append(file.ManagedResources, cfg)
//				}
//
//			case "data":
//				cfg, cfgDiags := decodeDataBlock(block)
//				diags = append(diags, cfgDiags...)
//				if cfg != nil {
//					file.DataResources = append(file.DataResources, cfg)
//				}
//
//			default:
//				// Should never happen because the above cases should be exhaustive
//				// for all block type names in our schema.
//				continue
//
//			}
//		}
//
//	}
//}


func isOverrideFile(filename string) bool {
	return false
}

//TODO handle overrides files

/*Terraform loads variables in the following order, with later sources taking precedence over earlier ones:

Environment variables
The terraform.tfvars file, if present.
The terraform.tfvars.json file, if present.
Any *.auto.tfvars or *.auto.tfvars.json files, processed in lexical order of their filenames.
Any -var and -var-file options on the command line, in the order they are provided. (This includes variables set by a Terraform Cloud workspace.)
*/
func (i *Interpreter) ParseHCLFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	// Just keep the file name, instead of the whole path
	_, filename = path.Split(filename)
	i.ParseHCL(data, filename)
}

func (i *Interpreter) ParseHCL(src []byte, filename string) {
	f, diags := i.parser.ParseHCL(src, filename)
	log.Printf("ok, %s, %s", f, diags)
}

func (i *Interpreter) ParseJSONFile(filename string) {
	log.Fatal("Not implemented")
}
