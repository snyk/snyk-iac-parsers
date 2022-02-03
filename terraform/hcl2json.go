package terraform

import (
	"encoding/json"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ExtractVariables extract the variables from the provided file
// When there are multiple files in a module, the variables get merged before being passed to the ParseHclToJson function
func ExtractVariables(fileName string, fileContent string) (VariableMap, error) {
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
