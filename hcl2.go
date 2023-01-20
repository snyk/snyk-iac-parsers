package parsers

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/snyk/snyk-iac-parsers/terraform"
)

// ParseHCL2 unmarshals HCL files that are written using
// version 2 of the HCL language and return parsed file content.
func ParseHCL2(p []byte, v interface{}) (err error) {
	result := terraform.ParseModule(map[string]interface{}{
		"foo.tf": string(p),
	})

	parsedFiles, ok := result["parsedFiles"]
	if !ok {
		return errors.Errorf("no parsed files returned")
	}

	parsedFilesMap, ok := parsedFiles.(map[string]interface{})
	if !ok {
		return errors.Errorf("invalid parsed files format")
	}

	parsed, ok := parsedFilesMap["foo.tf"]
	if !ok {
		return errors.Errorf("parse file")
	}

	parsedString, ok := parsed.(string)
	if !ok {
		return errors.Errorf("invalid parse result type")
	}

	if err := json.Unmarshal([]byte(parsedString), v); err != nil {
		return errors.Errorf("unmarshal parse result: %v", err)
	}

	return nil
}
