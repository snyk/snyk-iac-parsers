# snyk-iac-parsers
---

[![CircleCI](https://circleci.com/gh/snyk/snyk-iac-parsers/tree/main.svg?style=svg&circle-token=fc5da6b1544139b067e9d252270a60213a43e0d5)](https://circleci.com/gh/snyk/snyk-iac-parsers/tree/main)
[![Regression Tests](https://github.com/snyk/snyk-iac-parsers/actions/workflows/policies.yml/badge.svg)](https://img.shields.io/github/workflow/status/snyk/snyk-iac-parsers/Policies%20Regression%20Test?label=regression%20tests)

[![Latest release version](https://img.shields.io/github/v/release/snyk/snyk-iac-parsers)](https://github.com/snyk/snyk-iac-parsers)
[![Latest release date](https://img.shields.io/github/release-date/snyk/snyk-iac-parsers)](https://github.com/snyk/snyk-iac-rules)

[![Golang Version](https://img.shields.io/github/go-mod/go-version/snyk/snyk-iac-parsers)](https://github.com/snyk/snyk-iac-parsers)
---

This project includes parsers that are used for Snyk Infrastructure As Code product. Parsers convert the files they take as input into JSON. 

## Supported formats

The following file formats are supported:
- HCL2: [Terraform](https://www.terraform.io/)'s default configuration format, parser's source can be found [here](http://https://github.com/snyk/snyk-iac-parsers/blob/main/pkg/hcl2.go).
- Terraform Plan(JSON): [Terraform plan output in json](https://www.terraform.io/docs/internals/json-format.html) is parsed and ``resource_changes`` element is extracted. Parser's source can be found [here](https://github.com/snyk/snyk-iac-parsers/blob/main/pkg/terraform_plan.go).
- YAML: Parser's source can be found [here](https://github.com/snyk/snyk-iac-parsers/blob/main/pkg/yaml.go).

All the formats above are transformed into JSON so that they can be used as input into tools such as [Open Policy Agent](https://www.openpolicyagent.org/). 

## Development

Tests can be run using the `go test` command:

```bash
% go test ./...
```

Before committing code should be formatted with `go fmt` and linted with `golangci-lint run`. The CircleCI runner will enforce this for each opened pull request.

## Contributing

This project is developed in open as a dependency of the [snyk/snyk-iac-rules](https://github.com/snyk/snyk-iac-rules) project. Should you wish to make a contribution please open a pull request against this repository with a clear description of the change with tests demonstrating the functionality. You will also need to agree to the [Contributor Agreement](./Contributor-Agreement.md) before the code can be accepted and merged.

## License

Available under the [Mozilla Public License 2.0](./LICENSE.md)
