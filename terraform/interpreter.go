package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
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

	return interpreter
}

type rawFlag struct {
	Name  string
	Value string
}
type variableMap map[string]cty.Value

func (i *Interpreter) ModuleAsJson(dir string, env []string, rawFlags []rawFlag) ([]byte, error) {
	files := ParseTopLevelFilesInDirectory(i.parser, dir)
	i.TerraformModule = BuildModule(dir, files, DetectVarFiles(rawFlags)...)
	variables, _ := ParseInputVariables(i.TerraformModule, env, rawFlags)
	//TODO handle diags

	i.TerraformModule.vars = i.TerraformModule.MergeVariableValues(variables)
	resolveModuleCallVars(i.TerraformModule, i.TerraformModule.vars)

	return Convert(i.TerraformModule, Options{Simplify: true})
}

func resolveModuleCallVars(terraformModule *TerraformModule, vars variableMap) {
	for _, moduleCall := range terraformModule.moduleCalls {
		varMap := make(variableMap)
		for varName, expr := range moduleCall.InputValues {
			context := (&evalContext).NewChild()
			context.Variables = vars
			value, err := expr.Value(context)
			if err != nil {
				log.Printf("Err while parsing module call var: %s", err)
			} else {
				varMap[varName] = value
			}
		}
		terraformModule.childModules[moduleCall.Name].vars = createVarMap(make(variableMap), varMap)
	}
}

func ParseTopLevelFilesInDirectory(parser *hclparse.Parser, dir string) map[string]*hcl.File {
	ret := make(map[string]*hcl.File, 0)
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
			ret[file.Name()] = parseHCLFile(parser, filepath.Join(dir, file.Name()))
		} else if strings.HasSuffix(file.Name(), JSON) {
			//TODO
			//i.ParseJSONFile(filepath.Join(dir, file.Name()))
		}

	}
	return ret
}
func DetectVarFiles(args []rawFlag) []string {
	ret := make([]string, 0)
	for _, arg := range args {
		if arg.Value == ARG_VAR_FILE {
			ret = append(ret, arg.Value)
		}
	}
	return ret
}

func contains(list []string, term string) bool {
	for _, s := range list {
		if s == term {
			return true
		}
	}
	return false
}

func BuildModule(dir string, files map[string]*hcl.File, varFiles ...string) *TerraformModule {
	module := &TerraformModule{dir: dir, childModules: make(map[string]*TerraformModule)}
	for filename, hclFile := range files {
		var bodyContent *hcl.BodyContent
		var diags hcl.Diagnostics
		if strings.HasSuffix(filename, TF_VARS) || strings.HasSuffix(filename, TF_VARS_JSON) || contains(varFiles, filename) {
			bodyContent, _, _ = hclFile.Body.PartialContent(&hcl.BodySchema{
				Blocks: []hcl.BlockHeaderSchema{
					{
						Type:       "variable",
						LabelNames: []string{"name"},
					},
				},
			})
			addFileToModule(module, filename, hclFile, bodyContent, false)
		} else {
			bodyContent, diags = hclFile.Body.Content(configFileSchema)
			//TODO There might be var files with none standard name
			handleDiagnostics("Validation issue", diags, filename)
			addFileToModule(module, filename, hclFile, bodyContent, true)
		}

	}

	//Process modules
	for _, moduleCall := range module.moduleCalls {
		//if module count is zero, no need to process the module
		if v, ok := moduleCall.Count.(*hclsyntax.LiteralValueExpr); ok {
			intVal := ValueToInt(v.Val)
			if intVal != nil && *intVal == 0 {
				continue
			}
		}
		log.Printf("Module source: %s", moduleCall.SourceAddrRaw)
		//TODO handle vars
		normalizedPath := filepath.Join(dir, moduleCall.SourceAddrRaw)

		files := ParseTopLevelFilesInDirectory(hclparse.NewParser(), normalizedPath)
		module.childModules[moduleCall.Name] = BuildModule(normalizedPath, files)
	}

	return module
}

func ValueToInt(val cty.Value) *int {
	if val.Type() == cty.Number {
		b := val.AsBigFloat()
		if b.IsInt() {
			i64, _ := b.Int64()
			i := int(i64)
			return &i
		}
	}
	return nil
}

func addFileToModule(module *TerraformModule, filename string, hclFile *hcl.File, bodyContent *hcl.BodyContent, isConfig bool) {
	if !isOverrideFile(filename) {
		module.addFile(hclFile, bodyContent, filename, isConfig)
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

func parseHCLFile(parser *hclparse.Parser, filename string) *hcl.File {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	// Just keep the file name, instead of the whole path
	_, filename = path.Split(filename)
	return ParseHCL(parser, data, filename)
}

func ParseHCL(parser *hclparse.Parser, src []byte, filename string) *hcl.File {
	ret, diags := parser.ParseHCL(src, filename)
	handleDiagnostics("Parsing issue", diags, filename)
	return ret
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

func (i *Interpreter) ParseJSON(src []byte, filename string) *hcl.File {
	ret, diags := i.parser.ParseJSON(src, filename)
	handleDiagnostics("Parsing issue", diags, filename)
	return ret
}
