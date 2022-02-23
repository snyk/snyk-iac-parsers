package terraform

import (
	"encoding/json"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfFiles "github.com/snyk/snyk-iac-parsers/terraform/files"
	"github.com/snyk/snyk-iac-parsers/terraform/variables"
)

// ParseModule iterated through all the provided files in a module (.tf, terraform.tfvars, and *.auto.tfvars files)
// It extracts the variables from each one, merges them, and dereferences them one by one
func ParseModule(files map[string]interface{}) map[string]interface{} {
	failedFiles := make(map[string]interface{}) // will contain files alongside user errors
	parsedFiles := make(map[string]interface{})
	debugLogs := make(map[string]interface{})

	variablesMaps := map[string]variables.VariableMap{}
	for fileName, fileContentInterface := range files {
		if tfFiles.IsValidVariableFile(fileName) {
			// need to use interface{} for gopherjs, so cast it back to string
			fileContent, ok := fileContentInterface.(string)
			if !ok {
				continue
			}
			variableMap, err := extractVariables(fileName, fileContent)
			if err != nil {
				// skip non-user errors
				if isUserError(err) {
					debugLogs[fileName] = GenerateDebugLogs(err)
					failedFiles[fileName] = err.Error()
				}
				continue
			}
			variablesMaps[fileName] = variableMap
		}
	}

	// merge variables so they can be used across multiple files
	variableMap := variables.MergeVariables(variablesMaps)

	for fileName, fileContent := range files {
		// failedFiles contains user errors so if the file failed at extract time, we don't try to parse it
		if tfFiles.IsValidTerraformFile(fileName) && failedFiles[fileName] == nil {
			parsedJson, err := parseHclToJson(fileName, fileContent.(string), variableMap)
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

// extractVariables extracts the variables from the provided file
var extractVariables = func(fileName string, fileContent string) (variables.VariableMap, error) {
	file, diagnostics := hclsyntax.ParseConfig([]byte(fileContent), fileName, hcl.Pos{Line: 1, Column: 1})
	if diagnostics.HasErrors() {
		return variables.VariableMap{}, createInvalidHCLError(diagnostics.Errs())
	}

	vars, diagnostics := variables.ExtractFromFile(fileName, file)
	if diagnostics.HasErrors() {
		return variables.VariableMap{}, createInvalidHCLError(diagnostics.Errs())
	}

	return vars, nil
}

// ParseHclToJson parses a provided HCL file to JSON and dereferences any known variables using the provided variables
func ParseHclToJson(fileName string, fileContent string, vars variables.VariableMap) (string, error) {
	file, diagnostics := hclsyntax.ParseConfig([]byte(fileContent), fileName, hcl.Pos{Line: 1, Column: 1})
	if diagnostics.HasErrors() {
		return "", createInvalidHCLError(diagnostics.Errs())
	}

	parsedFile, err := parseFile(file, vars)
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
