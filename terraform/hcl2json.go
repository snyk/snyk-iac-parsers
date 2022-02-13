package terraform

import (
	"encoding/json"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ParseModule iterated through all the provided files in a module
// It extracts the variables from each one, merges them, and dereferences them one by one
func ParseModule(files map[string]interface{}) map[string]interface{} {
	failedFiles := make(map[string]interface{}) // will contain files alongside user errors
	parsedFiles := make(map[string]interface{})

	variablesMaps := []VariableMap{}
	for fileName, fileContentInterface := range files {
		// need to use interface{} for gopherjs, so cast it back to string
		fileContent, ok := fileContentInterface.(string)
		if !ok {
			continue
		}
		variableMap, err := extractVariables(fileName, fileContent)
		if err != nil {
			// skip non-user errors
			if isUserError(err) {
				failedFiles[fileName] = err.Error()
			}
			continue
		}
		variablesMaps = append(variablesMaps, variableMap)
	}

	// merge variables so they can be used across multiple files
	variableMap := mergeVariables(variablesMaps)

	for fileName, fileContent := range files {
		// failedFiles contains user errors so if the file failed at extract time, we don't try to parse it
		if failedFiles[fileName] == nil {
			parsedJson, err := parseHclToJson(fileName, fileContent.(string), variableMap)
			if err != nil {
				// skip non-user errors
				if isUserError(err) {
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
	}
}

// extractVariables extracts the variables from the provided file
var extractVariables = func(fileName string, fileContent string) (VariableMap, error) {
	file, diagnostics := hclsyntax.ParseConfig([]byte(fileContent), fileName, hcl.Pos{Line: 1, Column: 1})
	if diagnostics.HasErrors() {
		return VariableMap{}, createInvalidHCLError(diagnostics.Errs())
	}

	variables, diagnostics := extractFromFile(file)
	if diagnostics.HasErrors() {
		return VariableMap{}, createInvalidHCLError(diagnostics.Errs())
	}

	return variables, nil
}

// ParseHclToJson parses a provided HCL file to JSON and dereferences any known variables using the provided variables
func ParseHclToJson(fileName string, fileContent string, variables VariableMap) (string, error) {
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
