package terraform

import (
	"fmt"
	"strings"

	"github.com/zclconf/go-cty/cty"
	ctyconvert "github.com/zclconf/go-cty/cty/convert"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Parsing logic was taken from https://github.com/tmccombs/hcl2json/tree/a80b1cd24d787567ec3e93b6806077b8d6ee4d3d/convert

type Options struct {
	Simplify bool
}

type Parser struct {
	bytes     []byte
	variables ValueMap
	options   Options
}

type NewParserParams struct {
	bytes     []byte
	variables ModuleVariables
	options   Options
}

type JSON = map[string]interface{}

func NewParser(params NewParserParams) Parser {
	return Parser{
		bytes:     params.bytes,
		variables: createValueMap(params.variables),
		options:   params.options,
	}
}

func parseFile(file *hcl.File, variables ModuleVariables) (JSON, error) {
	parser := NewParser(NewParserParams{
		bytes:     file.Bytes,
		variables: variables,
		options: Options{
			Simplify: true,
		}})

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("Failed to parse hcl.Body to hclsyntax.Body type")
	}

	out := make(JSON)
	err := parser.parseBody(body, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (parser *Parser) parseBody(body *hclsyntax.Body, out JSON) error {
	var err error
	for key, value := range body.Attributes {
		out[key], err = parser.parseExpression(value.Expr)
		if err != nil {
			return fmt.Errorf("Failed to parse expression: %w", err)
		}
	}

	for _, block := range body.Blocks {
		if err := parser.parseBlock(block, out); err != nil {
			return fmt.Errorf("Failed to parse block: %w", err)
		}
	}

	return nil
}

func (parser *Parser) parseBlock(block *hclsyntax.Block, out JSON) error {
	key, nestedOut, err := parser.parseLabels(block.Type, block.Labels, out)
	if err != nil {
		return err
	}

	value := make(JSON)
	err = parser.parseBody(block.Body, value)
	if err != nil {
		return err
	}

	if current, exists := nestedOut[key]; exists {
		if list, ok := current.([]interface{}); ok {
			nestedOut[key] = append(list, value)
		} else {
			nestedOut[key] = []interface{}{current, value}
		}
	} else {
		nestedOut[key] = value
	}

	return nil
}

func (parser *Parser) parseLabels(key string, labels []string, out JSON) (string, JSON, error) {
	for _, label := range labels {
		// Checks to see if the label exists in the current output
		// When the label exists, move onto the next label reference.
		// When a label does not exist, create the label in the output and set that as the next label reference
		// in order to append (potential) labels to it.
		if _, exists := out[key]; exists {
			var ok bool
			out, ok = out[key].(JSON)
			if !ok {
				return "", nil, fmt.Errorf("Failed to convert block to JSON: %v.%v", key, strings.Join(labels, "."))
			}
		} else {
			out[key] = make(JSON)
			out = out[key].(JSON)
		}

		key = label
	}

	return key, out, nil
}

func (parser *Parser) parseExpression(expr hclsyntax.Expression) (interface{}, error) {
	if parser.options.Simplify {
		ctx := &hcl.EvalContext{
			Functions: terraformFunctions,
			Variables: parser.variables,
		}
		value, err := expr.Value(ctx)
		if err == nil {
			return ctyjson.SimpleJSONValue{Value: value}, nil
		}
	}

	switch value := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		return ctyjson.SimpleJSONValue{Value: value.Val}, nil
	case *hclsyntax.UnaryOpExpr:
		return parser.parseUnary(value)
	case *hclsyntax.TemplateExpr:
		return parser.parseTemplate(value)
	case *hclsyntax.TemplateWrapExpr:
		return parser.parseExpression(value.Wrapped)
	case *hclsyntax.TupleConsExpr:
		var list []interface{}
		for _, expr := range value.Exprs {
			elem, err := parser.parseExpression(expr)
			if err != nil {
				return nil, err
			}
			list = append(list, elem)
		}
		return list, nil
	case *hclsyntax.ObjectConsExpr:
		out := make(JSON)
		for _, item := range value.Items {
			key, err := parser.parseKey(item.KeyExpr)
			if err != nil {
				return nil, err
			}
			out[key], err = parser.parseExpression(item.ValueExpr)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	default:
		return parser.wrapExpr(expr), nil
	}
}

func (parser *Parser) parseUnary(v *hclsyntax.UnaryOpExpr) (interface{}, error) {
	_, isLiteral := v.Val.(*hclsyntax.LiteralValueExpr)
	if !isLiteral {
		// If the expression after the operator isn't a literal, fall back to
		// wrapping the expression with ${...}
		return parser.wrapExpr(v), nil
	}
	val, err := v.Value(nil)
	if err != nil {
		return nil, err
	}
	return ctyjson.SimpleJSONValue{Value: val}, nil
}

func (parser *Parser) parseTemplate(t *hclsyntax.TemplateExpr) (string, error) {
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
		s, err := parser.parseStringPart(part)
		if err != nil {
			return "", err
		}
		builder.WriteString(s)
	}
	return builder.String(), nil
}

func (parser *Parser) parseStringPart(expr hclsyntax.Expression) (string, error) {
	switch v := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		s, err := ctyconvert.Convert(v.Val, cty.String)
		if err != nil {
			return "", err
		}
		return s.AsString(), nil
	case *hclsyntax.TemplateExpr:
		return parser.parseTemplate(v)
	case *hclsyntax.TemplateWrapExpr:
		return parser.parseStringPart(v.Wrapped)
	case *hclsyntax.ConditionalExpr:
		return parser.parseTemplateConditional(v)
	case *hclsyntax.TemplateJoinExpr:
		return parser.parseTemplateFor(v.Tuple.(*hclsyntax.ForExpr))
	default:
		// treating as an embedded expression
		return parser.wrapExpr(expr), nil
	}
}

func (parser *Parser) parseKey(keyExpr hclsyntax.Expression) (string, error) {
	// a key should never have dynamic input
	if k, isKeyExpr := keyExpr.(*hclsyntax.ObjectConsKeyExpr); isKeyExpr {
		keyExpr = k.Wrapped
		if _, isTraversal := keyExpr.(*hclsyntax.ScopeTraversalExpr); isTraversal {
			return parser.rangeSource(keyExpr.Range()), nil
		}
	}
	return parser.parseStringPart(keyExpr)
}

func (parser *Parser) parseTemplateConditional(expr *hclsyntax.ConditionalExpr) (string, error) {
	var builder strings.Builder
	builder.WriteString("%{if ")
	builder.WriteString(parser.rangeSource(expr.Condition.Range()))
	builder.WriteString("}")
	trueResult, err := parser.parseStringPart(expr.TrueResult)
	if err != nil {
		return "", nil
	}
	builder.WriteString(trueResult)
	falseResult, err := parser.parseStringPart(expr.FalseResult)
	// TODO: do we need this?
	if err != nil {
		return "", nil
	}
	if len(falseResult) > 0 {
		builder.WriteString("%{else}")
		builder.WriteString(falseResult)
	}
	builder.WriteString("%{endif}")

	return builder.String(), nil
}

func (parser *Parser) parseTemplateFor(expr *hclsyntax.ForExpr) (string, error) {
	var builder strings.Builder
	builder.WriteString("%{for ")
	if len(expr.KeyVar) > 0 {
		builder.WriteString(expr.KeyVar)
		builder.WriteString(", ")
	}
	builder.WriteString(expr.ValVar)
	builder.WriteString(" in ")
	builder.WriteString(parser.rangeSource(expr.CollExpr.Range()))
	builder.WriteString("}")
	template, err := parser.parseStringPart(expr.ValExpr)
	if err != nil {
		return "", err
	}
	builder.WriteString(template)
	builder.WriteString("%{endfor}")

	return builder.String(), nil
}

func (parser *Parser) wrapExpr(expr hclsyntax.Expression) string {
	return "${" + parser.rangeSource(expr.Range()) + "}"
}

func (parser *Parser) rangeSource(r hcl.Range) string {
	// for some reason the range doesn't include the ending paren, so
	// check if the next character is an ending paren, and include it if it is.
	end := r.End.Byte
	if end < len(parser.bytes) && parser.bytes[end] == ')' {
		end++
	}
	return string(parser.bytes[r.Start.Byte:end])
}
