package parsers

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// ParseJSON unmarshals JSON files and return parsed file content.
func ParseJSON(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return errors.Wrap(err, "unmarshal json")
	}

	return nil
}
