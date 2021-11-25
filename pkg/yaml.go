package parsers

import (
	"bytes"
	"context"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// ParseYAML unmarshals YAML files and return parsed file content.
func ParseYAML(_ context.Context, p []byte, v interface{}) error {
	subDocuments := separateSubDocuments(p)
	if len(subDocuments) > 1 {
		if err := unmarshalMultipleDocuments(subDocuments, v); err != nil {
			return errors.Wrap(err, "unmarshal multiple documents")
		}

		return nil
	}

	if err := yaml.Unmarshal(p, v); err != nil {
		return errors.Wrap(err, "unmarshal yaml")
	}

	return nil
}

func separateSubDocuments(data []byte) [][]byte {
	linebreak := "\n"
	if bytes.Contains(data, []byte("\r\n---\r\n")) {
		linebreak = "\r\n"
	}

	return bytes.Split(data, []byte(linebreak+"---"+linebreak))
}

func unmarshalMultipleDocuments(subDocuments [][]byte, v interface{}) error {
	var documentStore []interface{}
	for _, subDocument := range subDocuments {
		var documentObject interface{}
		if err := yaml.Unmarshal(subDocument, &documentObject); err != nil {
			return errors.Wrap(err, "unmarshal subdocument yaml")
		}

		documentStore = append(documentStore, documentObject)
	}

	yamlConfigBytes, err := yaml.Marshal(documentStore)
	if err != nil {
		return errors.Wrap(err, "marshal yaml document")
	}

	if err := yaml.Unmarshal(yamlConfigBytes, v); err != nil {
		return errors.Wrap(err, "unmarshal yaml")
	}

	return nil
}
