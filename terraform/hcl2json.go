package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type File struct {
	fileName    string
	fileContent string
	hclFile     *hcl.File
}

type ParseModuleResult struct {
	parsedFiles map[string]interface{}
	failedFiles map[string]interface{} // will contain files alongside user errors
	debugLogs   map[string]interface{}
}

// ParseModule iterates through all the provided files in a module (.tf, terraform.tfvars, and *.auto.tfvars files)
// It extracts the variables from each one, merges them, and dereferences them one by one
func ParseModule(rawFiles map[string]interface{}) map[string]interface{} {
	parseRes := &ParseModuleResult{
		failedFiles: make(map[string]interface{}),
		parsedFiles: make(map[string]interface{}),
		debugLogs:   make(map[string]interface{}),
	}

	files := processFiles(rawFiles, parseRes)

	vars := extractModuleVariables(files, parseRes)

	parseModuleFiles(files, vars, parseRes)

	return JSON{
		"parsedFiles": parseRes.parsedFiles,
		"failedFiles": parseRes.failedFiles,
		"debugLogs":   parseRes.debugLogs,
	}
}

func processFiles(rawFiles map[string]interface{}, parseRes *ParseModuleResult) map[string]File {
	files := make(map[string]File)

	for fileName, fileContentInterface := range rawFiles {
		fileContent, ok := fileContentInterface.(string)
		if !ok {
			continue
		}

		hclFile, hclDiags := hclsyntax.ParseConfig([]byte(fileContent), fileName, hcl.Pos{Line: 1, Column: 1})
		if hclDiags.HasErrors() {
			err := createInvalidHCLError(hclDiags.Errs())
			parseRes.debugLogs[fileName] = GenerateDebugLogs(err)
			parseRes.failedFiles[fileName] = err.Error()
			continue
		}

		files[fileName] = File{
			fileName:    fileName,
			fileContent: fileContent,
			hclFile:     hclFile,
		}
	}

	return files
}

func parseModuleFiles(files map[string]File, vars ModuleVariables, parseRes *ParseModuleResult) {
	for fileName, file := range files {
		// failedFiles contains user errors so if the file failed at extract time, we don't try to parse it
		if _, ok := parseRes.failedFiles[fileName]; isValidTerraformFile(fileName) && !ok {
			parsedJson, err := parseHclToJson(fileName, file.fileContent, vars)
			if err != nil {
				// skip non-user errors
				if isUserError(err) {
					parseRes.failedFiles[fileName] = err.Error()
				}
				// but still log them
				parseRes.debugLogs[fileName] = GenerateDebugLogs(err)
				continue
			}
			parseRes.parsedFiles[fileName] = parsedJson
		}
	}
}

func extractModuleVariables(files map[string]File, parseRes *ParseModuleResult) ModuleVariables {
	inputsByFile := InputVariablesByFile{}
	localExprsMap := ExpressionMap{}

	for fileName, file := range files {
		inputsMap, localsMap, err := extractVariables(file)
		if err != nil {
			// skip non-user errors
			if isUserError(err) {
				parseRes.debugLogs[fileName] = GenerateDebugLogs(err)
				parseRes.failedFiles[fileName] = err.Error()
			}
		}
		inputsByFile[fileName] = inputsMap
		for localName, localVal := range localsMap {
			localExprsMap[localName] = localVal
		}
	}

	// merge inputs so they can be prioritised and used across multiple files
	inputs := mergeInputVariables(inputsByFile)

	// dereference locals in case they reference each other or other input variables
	locals := dereferenceLocals(localExprsMap, inputs)

	return ModuleVariables{
		inputs: inputs,
		locals: locals,
	}
}

// used for mocking in the tests
var parseHclToJson = ParseHclToJson
var extractVariables = ExtractVariables
