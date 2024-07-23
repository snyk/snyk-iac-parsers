package terraform

import (
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// Function definitions were taken from https://github.com/tmccombs/hcl2json/tree/a80b1cd24d787567ec3e93b6806077b8d6ee4d3d/convert/stdlib.go

// a subset of functions used in terraform
// that can be used when simplifying during conversion
var terraformFunctions = map[string]function.Function{
	// numeric
	"abs":      stdlib.AbsoluteFunc,
	"ceil":     stdlib.CeilFunc,
	"floor":    stdlib.FloorFunc,
	"log":      stdlib.LogFunc,
	"max":      stdlib.MaxFunc,
	"min":      stdlib.MinFunc,
	"parseint": stdlib.ParseIntFunc,
	"pow":      stdlib.PowFunc,
	"signum":   stdlib.SignumFunc,

	// string
	"chomp":      stdlib.ChompFunc,
	"format":     stdlib.FormatFunc,
	"formatlist": stdlib.FormatListFunc,
	"indent":     stdlib.IndentFunc,
	"join":       stdlib.JoinFunc,
	"split":      stdlib.SplitFunc,
	"strrev":     stdlib.ReverseFunc,
	"trim":       stdlib.TrimFunc,
	"trimprefix": stdlib.TrimPrefixFunc,
	"trimsuffix": stdlib.TrimSuffixFunc,
	"trimspace":  stdlib.TrimSpaceFunc,

	// collections
	"chunklist": stdlib.ChunklistFunc,
	"concat":    stdlib.ConcatFunc,
	"distinct":  stdlib.DistinctFunc,
	"flatten":   stdlib.FlattenFunc,
	"length":    stdlib.LengthFunc,
	"merge":     stdlib.MergeFunc,
	"reverse":   stdlib.ReverseListFunc,
	"sort":      stdlib.SortFunc,

	// encoding
	"csvdecode":  stdlib.CSVDecodeFunc,
	"jsondecode": stdlib.JSONDecodeFunc,
	"jsonencode": stdlib.JSONEncodeFunc,

	// time
	"formatdate": stdlib.FormatDateFunc,
	"timeadd":    stdlib.TimeAddFunc,
}

// Set of functions which should not be allowed as their execution can be a
// security vulnerability.  Currently, this list is just the contents of the
// "Filesystem Functions" section in the TF docs:
// https://developer.hashicorp.com/terraform/language/functions/list
var disallowedTerraformFunctions = map[string]bool{
	"abspath":      true,
	"dirname":      true,
	"pathexpand":   true,
	"basename":     true,
	"file":         true,
	"fileexists":   true,
	"fileset":      true,
	"filebase64":   true,
	"templatefile": true,
}
