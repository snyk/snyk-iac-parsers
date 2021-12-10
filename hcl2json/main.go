package main

import (
	"github.com/gopherjs/gopherjs/js"
	"github.com/miratronix/jopher"

	parsers "github.com/snyk/snyk-iac-parsers"
)

type SDK struct {
	*js.Object
	Parse func(...interface{}) *js.Object `js:"parse"`
}

func (sdk *SDK) parse() (string, error) {
	return parsers.HCL2JSON()
}

func newHCL2JSONParser() *js.Object {
	sdk := SDK{Object: js.Global.Get("Object").New()}
	sdk.Parse = jopher.Promisify(sdk.parse)
	return sdk.Object
}

func main() {
	js.Module.Get("exports").Set("newHCL2JSONParser", newHCL2JSONParser)
}
