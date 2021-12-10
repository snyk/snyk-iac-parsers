package main

import (
	"github.com/gopherjs/gopherjs/js"
	"github.com/miratronix/jopher"

	parsers "github.com/snyk/snyk-iac-parsers"
)

type HCLJ2SONParser struct {
	*js.Object
	File       string                          `js:"file"`
	Path       string                          `js:"path"`
	Parse      func(...interface{}) *js.Object `js:"parse"`
	LineNumber func(...interface{}) *js.Object `js:"lineNumber"`
}

func (o *HCLJ2SONParser) parse() (string, error) {
	return parsers.HCL2JSON(o.File)
}

func (o *HCLJ2SONParser) lineNumber() (string, error) {
	return parsers.LineNumber(o.File, o.Path)
}

func newHCL2JSONParser(file string, path string) *js.Object {
	o := HCLJ2SONParser{Object: js.Global.Get("Object").New()}
	o.File = file
	o.Path = path
	o.Parse = jopher.Promisify(o.parse)
	o.LineNumber = jopher.Promisify(o.lineNumber)
	return o.Object
}

func main() {
	js.Module.Get("exports").Set("newHCL2JSONParser", newHCL2JSONParser)
	//	 module.exports = { newHCL2JSONParser: newHCL2JSONParser}
}
