package terraform

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type ValueMap map[string]cty.Value

type ExpressionMap map[string]hcl.Expression

type ModuleVariables struct {
	inputs ValueMap
	locals ValueMap
}

type InputVariablesByFile map[string]ExpressionMap

// ExtractVariables extracts the input variables and local values from the provided file
func ExtractVariables(file File) (ExpressionMap, ExpressionMap, error) {
	inputsMap := ExpressionMap{}
	localsMap := ExpressionMap{}
	var hclDiags hcl.Diagnostics

	if isValidInputVariablesFile(file.fileName) {
		var inputHclDiags hcl.Diagnostics
		inputsMap, hclDiags = extractInputVariablesFromFile(file)
		hclDiags = append(hclDiags, inputHclDiags...)
	}

	if isValidTerraformFile(file.fileName) {
		var localsHclDiags hcl.Diagnostics
		localsMap, localsHclDiags = extractLocalsFromFile(file)
		hclDiags = append(hclDiags, localsHclDiags...)
	}

	return inputsMap, localsMap, hclDiags
}

func extractInputVariablesFromFile(file File) (ExpressionMap, hcl.Diagnostics) {
	var inputVariables ExpressionMap
	var hclDiags hcl.Diagnostics
	if strings.HasSuffix(file.fileName, TF) {
		inputVariables, hclDiags = extractInputVariablesFromTfFile(file.hclFile)
	} else if strings.HasSuffix(file.fileName, TFVARS) {
		inputVariables, hclDiags = extractInputVariablesFromTfvarsFile(file.hclFile)
	}

	if hclDiags.HasErrors() {
		return inputVariables, hclDiags
	}

	return inputVariables, hclDiags
}

func extractLocalsFromFile(file File) (ExpressionMap, hcl.Diagnostics) {
	localExprsMap := ExpressionMap{}

	bodyContent, _, hclDiags := file.hclFile.Body.PartialContent(tfFileLocalSchema)
	if hclDiags.HasErrors() {
		return localExprsMap, hclDiags
	}

	for _, block := range bodyContent.Blocks {
		attrs, _ := block.Body.JustAttributes()
		for localName, attr := range attrs {
			localExprsMap[localName] = attr.Expr
		}
	}

	return localExprsMap, hclDiags
}

// Logic inspired from https://github.com/hashicorp/terraform/blob/f266d1ee82d1fa4d882c146cc131fec4bef753cf/internal/configs/named_values.go#L113
func extractInputVariablesFromTfFile(file *hcl.File) (ExpressionMap, hcl.Diagnostics) {
	inputVariablesMap := ExpressionMap{}

	bodyContent, _, hclDiags := file.Body.PartialContent(tfFileVariableSchema)
	if hclDiags.HasErrors() {
		return inputVariablesMap, hclDiags
	}

	for _, block := range bodyContent.Blocks {
		name := block.Labels[0]

		attrs, _ := block.Body.JustAttributes()
		defaultValue := attrs["default"]
		if defaultValue != nil {
			inputVariablesMap[name] = defaultValue.Expr
		}

	}

	return inputVariablesMap, hclDiags
}

func extractInputVariablesFromTfvarsFile(file *hcl.File) (ExpressionMap, hcl.Diagnostics) {
	inputVariablesMap := ExpressionMap{}

	attrs, hclDiags := file.Body.JustAttributes()
	for name, attr := range attrs {
		inputVariablesMap[name] = attr.Expr
	}

	return inputVariablesMap, hclDiags
}

func mergeInputVariables(inputVariablesByFile InputVariablesByFile) ExpressionMap {
	combinedInputVariables := make(ExpressionMap)

	fileNames := make([]string, 0, len(inputVariablesByFile))
	for fileName := range inputVariablesByFile {
		fileNames = append(fileNames, fileName)
	}

	prioritisedFileNames := orderFilesByPriority(fileNames)

	for _, fileName := range prioritisedFileNames {
		inputVariablesMap := inputVariablesByFile[fileName]
		for input, value := range inputVariablesMap {
			combinedInputVariables[input] = value
		}
	}

	return combinedInputVariables
}

var maxLocalsDerefIterations = 32

func dereferenceVariables(locals ExpressionMap, inputs ExpressionMap) ModuleVariables {
	currentVariables := ModuleVariables{
		inputs: ValueMap{},
		locals: ValueMap{},
	}
	nextVariables := ModuleVariables{
		inputs: ValueMap{},
		locals: ValueMap{},
	}

	for i := 0; i < maxLocalsDerefIterations; i++ {
		for inputName, inputExpr := range inputs {
			inputVal := dereferenceVariable(inputExpr, currentVariables)
			if !inputVal.IsNull() {
				nextVariables.inputs[inputName] = inputVal
			}
		}
		for localName, localExpr := range locals {
			localVal := dereferenceVariable(localExpr, currentVariables)
			if !localVal.IsNull() {
				nextVariables.locals[localName] = localVal
			}
		}

		// stop if the local values haven't changed between dereferencing loops
		if reflect.DeepEqual(currentVariables, nextVariables) {
			break
		}

		for inputName, inputVal := range nextVariables.inputs {
			currentVariables.inputs[inputName] = inputVal
		}

		for localName, localVal := range nextVariables.locals {
			currentVariables.locals[localName] = localVal
		}
	}

	return nextVariables
}

func dereferenceVariable(expr hcl.Expression, variables ModuleVariables) cty.Value {
	value, hclDiags := expr.Value(&hcl.EvalContext{
		Variables: createValueMap(variables),
		Functions: terraformFunctions,
	})

	// the variable cannot be dereferenced so move onto the next one
	if !value.IsKnown() || hclDiags.HasErrors() {
		return cty.NilVal
	}

	// a variable has been dereferenced so store it in the
	return value
}

func createValueMap(variables ModuleVariables) ValueMap {
	return ValueMap{
		"var":   cty.ObjectVal(variables.inputs),
		"local": cty.ObjectVal(variables.locals),
	}
}
