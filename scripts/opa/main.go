package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/cmd"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"

	parsers "github.com/snyk/snyk-iac-parsers"
	terraform "github.com/snyk/snyk-iac-parsers/terraform"
)

func runOpa() {
	rego.RegisterBuiltin1(
		&rego.Function{
			Name:    "hcl2.unmarshal_file",
			Decl:    types.NewFunction(types.Args(types.S), types.A),
			Memoize: true,
		},
		func(bctx rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {

			var filePath string

			if err := ast.As(a.Value, &filePath); err != nil {
				return nil, err
			}

			content, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, err
			}

			input, err := terraform.ParseHclToJson(filePath, string(content), terraform.ModuleVariables{})
			if err != nil {
				return nil, err
			}

			var parsedInput terraform.JSON
			err = json.Unmarshal([]byte(input), &parsedInput)
			if err != nil {
				return nil, err
			}

			v, err := ast.InterfaceToValue(parsedInput)
			if err != nil {
				return nil, err
			}

			return ast.NewTerm(v), nil
		},
	)

	rego.RegisterBuiltin1(
		&rego.Function{
			Name:    "yaml.unmarshal_file",
			Decl:    types.NewFunction(types.Args(types.S), types.A),
			Memoize: true,
		},
		func(bctx rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {

			var filePath string

			if err := ast.As(a.Value, &filePath); err != nil {
				return nil, err
			}

			content, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, err
			}

			var parsedInput interface{}
			if err := parsers.ParseYAML(content, &parsedInput); err != nil {
				return nil, err
			}
			v, err := ast.InterfaceToValue(parsedInput)
			if err != nil {
				return nil, err
			}

			return ast.NewTerm(v), nil
		},
	)

	if err := cmd.RootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	runOpa()
}
