package terraform

import (
	"encoding/json"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ParseModule iterated through all the provided files in a module (.tf, terraform.tfvars, and *.auto.tfvars files)
// It extracts the variables from each one, merges them, and dereferences them one by one
func ParseModule(files map[string]interface{}) map[string]interface{} {
	failedFiles := make(map[string]interface{}) // will contain files alongside user errors
	parsedFiles := make(map[string]interface{})
	debugLogs := make(map[string]interface{})

	inputsByFile := InputsByFile{}
	for fileName, fileContentInterface := range files {
		if isValidInputsFile(fileName) {
			// need to use interface{} for gopherjs, so cast it back to string
			fileContent, ok := fileContentInterface.(string)
			if !ok {
				continue
			}
			inputsMap, err := extractInputs(fileName, fileContent)
			if err != nil {
				// skip non-user errors
				if isUserError(err) {
					debugLogs[fileName] = GenerateDebugLogs(err)
					failedFiles[fileName] = err.Error()
				}
				continue
			}
			inputsByFile[fileName] = inputsMap
		}
	}

	// merge inputs so they can be used across multiple files
	inputsMap := mergeInputs(inputsByFile)

	vars := ModuleVariables{
		inputs: inputsMap,
	}

	for fileName, fileContent := range files {
		// failedFiles contains user errors so if the file failed at extract time, we don't try to parse it
		if isValidTerraformFile(fileName) && failedFiles[fileName] == nil {
			parsedJson, err := parseHclToJson(fileName, fileContent.(string), vars)
			if err != nil {
				// skip non-user errors
				if isUserError(err) {
					debugLogs[fileName] = GenerateDebugLogs(err)
					failedFiles[fileName] = err.Error()
				}
				continue
			}
			parsedFiles[fileName] = parsedJson
		}
	}

	return map[string]interface{}{
		"parsedFiles": parsedFiles,
		"failedFiles": failedFiles,
		"debugLogs":   debugLogs,
	}
}

// extractInputs extracts the input values from the provided file
var extractInputs = func(fileName string, fileContent string) (ValueMap, error) {
	file, diagnostics := hclsyntax.ParseConfig([]byte(fileContent), fileName, hcl.Pos{Line: 1, Column: 1})
	if diagnostics.HasErrors() {
		return ValueMap{}, createInvalidHCLError(diagnostics.Errs())
	}

	fileInputsMap, diagnostics := extractInputsFromFile(fileName, file)
	if diagnostics.HasErrors() {
		return ValueMap{}, createInvalidHCLError(diagnostics.Errs())
	}

	return fileInputsMap, nil
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

var parseHclToJson = ParseHclToJson // used for mocking in the tests
