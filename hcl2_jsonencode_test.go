package parsers_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	parsers "github.com/snyk/snyk-iac-parsers"
)

func TestParseHCL2WithJSONEncode(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("testdata", "terraform", "jsonencode.tf"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var result interface{}

	if err := parsers.ParseHCL2([]byte(input), &result); err != nil {
		t.Fatalf("parse HCL2: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal parsed input: %v", err)
	}

	if strings.Contains(string(data), "jsonencode") {
		t.Fatalf("jsonencode has not been evaluated")
	}
}
