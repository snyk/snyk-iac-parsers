package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"log"
)

type TerraformModule struct {
	Files      []*TerraformFile
	blocks     hcl.Blocks
	attributes hcl.Attributes
	variables  []Variable
	locals     []Local
}

type TerraformFile struct {
	File        *hcl.File
	BodyContent *hcl.BodyContent
	filename    string
	variables   []Variable
	locals      []Local
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
			default:
				continue
			}
		}
	}
}
