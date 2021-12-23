package terraform

import (
	"encoding/json"
	"fmt"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	ctyconvert "github.com/zclconf/go-cty/cty/convert"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

type variableMap map[string]cty.Value

type Options struct {
	Simplify    bool
	ContextVars HclEvalContextVars
}

type HclEvalContextVars struct {
	Val variableMap
}

func NewHclEvalContextVars() HclEvalContextVars {
	return HclEvalContextVars{Val: make(variableMap)}
}

func (h *HclEvalContextVars) addVars(vars variableMap) {
	h.Val["var"] = cty.ObjectVal(vars)
}

func (h *HclEvalContextVars) addLocals(locals variableMap) {
	h.Val["local"] = cty.ObjectVal(locals)
}

func Convert(module *TerraformModule, options Options) ([]byte, error) {
	convertedFile, err := convertFiles(module, options)
	if err != nil {
		return nil, fmt.Errorf("convert file: %w", err)
	}

	fileBytes, err := json.MarshalIndent(convertedFile, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	return fileBytes, nil
}

type jsonObj map[string]interface{}

func convertFiles(module *TerraformModule, options Options) (jsonObj, error) {
	c := converter{module: module, options: options}
	out := make(jsonObj)
	for _, file := range module.Files {
		if file.isConfig {
			body := file.File.Body.(*hclsyntax.Body)

			c.convertBody(body, file.File, out)
		}
	}

	return out, nil
}

type converter struct {
	module  *TerraformModule
	options Options
}

func (c *converter) rangeSource(r hcl.Range, file *hcl.File) string {
	// for some reason the range doesn't include the ending paren, so
	// check if the next character is an ending paren, and include it if it is.
	end := r.End.Byte
	if file.Bytes[end] == ')' {
		end++
	}
	return string(file.Bytes[r.Start.Byte:end])
}

func (c *converter) convertBody(body *hclsyntax.Body, file *hcl.File, out jsonObj) (jsonObj, error) {
	var err error
	if out == nil {
		out = make(jsonObj)
	}
	for key, value := range body.Attributes {
		out[key], err = c.convertExpression(value.Expr, file)
		if err != nil {
			return nil, err
		}
	}

	for _, block := range body.Blocks {
		err = c.convertBlock(block, out, file)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (c *converter) convertBlock(block *hclsyntax.Block, out jsonObj, file *hcl.File) error {
	var key string = block.Type

	value, err := c.convertBody(block.Body, file, nil)
	if err != nil {
		return err
	}

	for _, label := range block.Labels {
		if inner, exists := out[key]; exists {
			var ok bool
			out, ok = inner.(jsonObj)
			if !ok {
				// TODO: better diagnostics
				return fmt.Errorf("Unable to convert Block to JSON: %v.%v", block.Type, strings.Join(block.Labels, "."))
			}
		} else {
			obj := make(jsonObj)
			out[key] = obj
			out = obj
		}
		key = label
	}

	if current, exists := out[key]; exists {
		if list, ok := current.([]interface{}); ok {
			out[key] = append(list, value)
		} else {
			out[key] = []interface{}{current, value}
		}
	} else {
		out[key] = value
	}

	return nil
}

func (c *converter) convertExpression(expr hclsyntax.Expression, file *hcl.File) (interface{}, error) {
	if c.options.Simplify {
		context := (&evalContext).NewChild()
		context.Variables = c.options.ContextVars.Val
		value, err := expr.Value(context)
		if err == nil {
			return ctyjson.SimpleJSONValue{Value: value}, nil
		}
	}
	// assume it is hcl syntax (because, um, it is)
	switch value := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		return ctyjson.SimpleJSONValue{Value: value.Val}, nil
	case *hclsyntax.UnaryOpExpr:
		return c.convertUnary(value, file)
	case *hclsyntax.TemplateExpr:
		return c.convertTemplate(value, file)
	case *hclsyntax.TemplateWrapExpr:
		return c.convertExpression(value.Wrapped, file)
	case *hclsyntax.TupleConsExpr:
		var list []interface{}
		for _, ex := range value.Exprs {
			elem, err := c.convertExpression(ex, file)
			if err != nil {
				return nil, err
			}
			list = append(list, elem)
		}
		return list, nil
	case *hclsyntax.ObjectConsExpr:
		m := make(jsonObj)
		for _, item := range value.Items {
			key, err := c.convertKey(item.KeyExpr, file)
			if err != nil {
				return nil, err
			}
			m[key], err = c.convertExpression(item.ValueExpr, file)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	default:
		return c.wrapExpr(expr, file), nil
	}
}

func (c *converter) convertUnary(v *hclsyntax.UnaryOpExpr, file *hcl.File) (interface{}, error) {
	_, isLiteral := v.Val.(*hclsyntax.LiteralValueExpr)
	if !isLiteral {
		// If the expression after the operator isn't a literal, fall back to
		// wrapping the expression with ${...}
		return c.wrapExpr(v, file), nil
	}
	val, err := v.Value(nil)
	if err != nil {
		return nil, err
	}
	return ctyjson.SimpleJSONValue{Value: val}, nil
}

func (c *converter) convertTemplate(t *hclsyntax.TemplateExpr, file *hcl.File) (string, error) {
	if t.IsStringLiteral() {
		// safe because the value is just the string
		v, err := t.Value(nil)
		if err != nil {
			return "", err
		}
		return v.AsString(), nil
	}
	var builder strings.Builder
	for _, part := range t.Parts {
		s, err := c.convertStringPart(part, file)
		if err != nil {
			return "", err
		}
		builder.WriteString(s)
	}
	return builder.String(), nil
}

func (c *converter) convertStringPart(expr hclsyntax.Expression, file *hcl.File) (string, error) {
	switch v := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		s, err := ctyconvert.Convert(v.Val, cty.String)
		if err != nil {
			return "", err
		}
		return s.AsString(), nil
	case *hclsyntax.TemplateExpr:
		return c.convertTemplate(v, file)
	case *hclsyntax.TemplateWrapExpr:
		return c.convertStringPart(v.Wrapped, file)
	case *hclsyntax.ConditionalExpr:
		return c.convertTemplateConditional(v, file)
	case *hclsyntax.TemplateJoinExpr:
		return c.convertTemplateFor(v.Tuple.(*hclsyntax.ForExpr), file)
	default:
		// treating as an embedded expression
		return c.wrapExpr(expr, file), nil
	}
}

func (c *converter) convertKey(keyExpr hclsyntax.Expression, file *hcl.File) (string, error) {
	// a key should never have dynamic input
	if k, isKeyExpr := keyExpr.(*hclsyntax.ObjectConsKeyExpr); isKeyExpr {
		keyExpr = k.Wrapped
		if _, isTraversal := keyExpr.(*hclsyntax.ScopeTraversalExpr); isTraversal {
			return c.rangeSource(keyExpr.Range(), file), nil
		}
	}
	return c.convertStringPart(keyExpr, file)
}

func (c *converter) convertTemplateConditional(expr *hclsyntax.ConditionalExpr, file *hcl.File) (string, error) {
	var builder strings.Builder
	builder.WriteString("%{if ")
	builder.WriteString(c.rangeSource(expr.Condition.Range(), file))
	builder.WriteString("}")
	trueResult, err := c.convertStringPart(expr.TrueResult, file)
	if err != nil {
		return "", nil
	}
	builder.WriteString(trueResult)
	falseResult, err := c.convertStringPart(expr.FalseResult, file)
	if len(falseResult) > 0 {
		builder.WriteString("%{else}")
		builder.WriteString(falseResult)
	}
	builder.WriteString("%{endif}")

	return builder.String(), nil
}

func (c *converter) convertTemplateFor(expr *hclsyntax.ForExpr, file *hcl.File) (string, error) {
	var builder strings.Builder
	builder.WriteString("%{for ")
	if len(expr.KeyVar) > 0 {
		builder.WriteString(expr.KeyVar)
		builder.WriteString(", ")
	}
	builder.WriteString(expr.ValVar)
	builder.WriteString(" in ")
	builder.WriteString(c.rangeSource(expr.CollExpr.Range(), file))
	builder.WriteString("}")
	templ, err := c.convertStringPart(expr.ValExpr, file)
	if err != nil {
		return "", err
	}
	builder.WriteString(templ)
	builder.WriteString("%{endfor}")

	return builder.String(), nil
}

func (c *converter) wrapExpr(expr hclsyntax.Expression, file *hcl.File) string {
	return "${" + c.rangeSource(expr.Range(), file) + "}"
}
