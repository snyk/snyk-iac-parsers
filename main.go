package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/tmccombs/hcl2json/convert"
)

func main() {
	filename := "test.tf"

	file, err := os.ReadFile(filename)
	if err != nil {
		fmt.Errorf(err.Error())
	}
	parsedFile, _ := hclsyntax.ParseConfig(file, filename, hcl.Pos{Line: 1, Column: 1})

	var options convert.Options = convert.Options{
		Simplify: false,
	}
	// TODO: still using the older version
	hclBytes, err := convert.File(parsedFile, options)
	if err != nil {
		fmt.Errorf("convert to HCL: %w", err)
	}

	// TODO: stretch item - actually use this in the code (gopherjs)
	fmt.Println(string(hclBytes))

	parsedFile.
}
