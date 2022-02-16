package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"log"
)

type TerraformModuleTree struct {
	main     *TerraformModule
	children []*TerraformModule
}

type TerraformModule struct {
	Files        []*TerraformFile
	blocks       hcl.Blocks
	attributes   hcl.Attributes
	variables    []Variable
	locals       []Local
	moduleCalls  []ModuleCall
	dir          string
	childModules map[string]*TerraformModule
}

type TerraformFile struct {
	File        *hcl.File
	BodyContent *hcl.BodyContent
	filename    string
	variables   []Variable
	locals      []Local
	moduleCalls []ModuleCall
	isConfig    bool
}

func (m *TerraformModule) addFile(hclFile *hcl.File, body *hcl.BodyContent, filename string, isConfigFile bool) {

	diags := hcl.Diagnostics{}
	terraformFile := &TerraformFile{File: hclFile, BodyContent: body, filename: filename, isConfig: isConfigFile}
	m.Files = append(m.Files, terraformFile)

	if isConfigFile {
		// naively append every block without any logic
		m.blocks = append(m.blocks, body.Blocks...)
		// Merge attribu†es in top level
		for key, attribute := range body.Attributes {
			if _, ok := m.attributes[key]; ok {
				//do something here
				log.Fatalf("Attribute: %s already exists", key)
			} else {
				m.attributes[key] = attribute
			}
		}

		//extract variables and locals
		for _, block := range body.Blocks {
			switch block.Type {
			case "variable":
				variable, cfgDiags := decodeVariableBlock(block, false)
				diags = append(diags, cfgDiags...)
				terraformFile.variables = append(terraformFile.variables, variable)
				m.variables = append(m.variables, variable)
			case "locals":
				locals, defsDiags := decodeLocalsBlock(block)
				diags = append(diags, defsDiags...)
				terraformFile.locals = append(terraformFile.locals, locals...)
				m.locals = append(m.locals, locals...)
			case "module":
				cfg, cfgDiags := decodeModuleBlock(block, false)
				diags = append(diags, cfgDiags...)
				terraformFile.moduleCalls = append(terraformFile.moduleCalls, cfg)
				m.moduleCalls = append(m.moduleCalls, cfg)

			default:
				continue
			}
		}
	}
}
func (m *TerraformModule) MergeVariables(inputs map[string]*InputValue) map[string]cty.Value {
	ret := make(variableMap)

	vars := make(variableMap)
	//Handle variable default values
	for _, variable := range m.variables {
		if variable.DefaultSet {
			vars[variable.Name] = variable.Default
		}
	}
	//Override variable defaults with input
	for name, inputValue := range inputs {
		vars[name] = inputValue.Value
	}
	ret["var"] = cty.ObjectVal(vars)

	locals := make(variableMap)

	//Handle locals
	for _, local := range m.locals {
		locals[local.Name] = local.Value
	}

	ret["local"] = cty.ObjectVal(locals)

	return ret
}
