package terraform

import (
	"encoding/json"

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

// ParseModule iterated through all the provided files in a module (.tf, terraform.tfvars, and *.auto.tfvars files)
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

// extractInputVariables extracts the input variables values from the provided file
var extractInputVariables = func(file File) (ValueMap, error) {
	fileInputVariablesMap, diagnostics := extractInputVariablesFromFile(file)
	if diagnostics.HasErrors() {
		return ValueMap{}, createInvalidHCLError(diagnostics.Errs())
	}

	return fileInputVariablesMap, nil
}

// extractLocals extracts the input values from the provided file
var extractLocals = func(file File, inputsMap ValueMap) (ValueMap, error) {
	localsMap, diagnostics := extractLocalsFromFile(file, inputsMap)
	if diagnostics.HasErrors() {
		return ValueMap{}, createInvalidHCLError(diagnostics.Errs())
	}

	return localsMap, nil
}

// ParseHclToJson parses a provided HCL file to JSON and dereferences any known variables using the provided variables
func ParseHclToJson(fileName string, fileContent string, variables ModuleVariables) (string, error) {
	file, diagnostics := hclsyntax.ParseConfig([]byte(fileContent), fileName, hcl.Pos{Line: 1, Column: 1})
	if diagnostics.HasErrors() {
		return "", createInvalidHCLError(diagnostics.Errs())
	}

	parsedFile, err := parseFile(file, variables)
	if err != nil {
		return "", createInternalHCLParsingError([]error{err})
	}

	jsonBytes, err := json.MarshalIndent(parsedFile, "", "\t")
	if err != nil {
		return "", createInternalJSONParsingError([]error{err})
	}

	return string(jsonBytes), nil
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
					parseRes.debugLogs[fileName] = GenerateDebugLogs(err)
					parseRes.failedFiles[fileName] = err.Error()
				}
				continue
			}
			parseRes.parsedFiles[fileName] = parsedJson
		}
	}
}

func extractModuleVariables(files map[string]File, parseRes *ParseModuleResult) ModuleVariables {
	inputsByFile := InputVariablesByFile{}

	for fileName, file := range files {
		if isValidInputVariablesFile(fileName) {
			inputsMap, err := extractInputVariables(file)
			if err != nil {
				// skip non-user errors
				if isUserError(err) {
					parseRes.debugLogs[fileName] = GenerateDebugLogs(err)
					parseRes.failedFiles[fileName] = err.Error()
				}
				continue
			}
			inputsByFile[fileName] = inputsMap
		}
	}

	// merge inputs so they can be used across multiple files
	inputsMap := mergeInputVariables(inputsByFile)

	localsMap := ValueMap{}

	for fileName, file := range files {
		if _, ok := parseRes.failedFiles[fileName]; !ok && isValidLocalsFile(fileName) {
			res, err := extractLocals(file, inputsMap)
			if err != nil {
				// skip non-user errors
				if isUserError(err) {
					parseRes.debugLogs[fileName] = GenerateDebugLogs(err)
					parseRes.failedFiles[fileName] = err.Error()
				}
				continue
			}

			for localName, localVal := range res {
				localsMap[localName] = localVal
			}
		}
	}

	return ModuleVariables{
		inputs: inputsMap,
		locals: localsMap,
	}
}

var parseHclToJson = ParseHclToJson // used for mocking in the tests
