package parsers

import (
	"encoding/json"
	"github.com/pkg/errors"
	hcl2jsonConverter "github.com/tmccombs/hcl2json/convert"
)

// ParseHCL2 unmarshals HCL files that are written using
// version 2 of the HCL language and return parsed file content.
func ParseHCL2(p []byte, v interface{}) (err error) {
	// TODO: Look into using plain github.com/hashicorp/hcl/v2
	// instead to avoid the JSON intermediary format.

	jsonBytes, err := hcl2jsonConverter.Bytes(p, "", hcl2jsonConverter.Options{})
	if err != nil {
		return errors.Wrap(err, "hcl2 to json conversion failed")
	}

	if err := json.Unmarshal(jsonBytes, v); err != nil {
		return errors.Wrap(err, "unmarshal hcl2 json failed")
	}

	return nil
}
