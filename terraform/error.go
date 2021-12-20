package terraform

import (
	"bytes"
	"fmt"
	"github.com/zclconf/go-cty/cty"
	"strconv"
)

// nativeError is a Diagnostic implementation that wraps a normal Go error
type nativeError struct {
	err error
}

var _ Diagnostic = nativeError{}

func (e nativeError) Severity() Severity {
	return Error
}

func (e nativeError) Description() Description {
	return Description{
		Summary: FormatError(e.err),
	}
}

func (e nativeError) Source() Source {
	// No source information available for a native error
	return Source{}
}

func (e nativeError) FromExpr() *FromExpr {
	// Native errors are not expression-related
	return nil
}
func FormatError(err error) string {
	perr, ok := err.(cty.PathError)
	if !ok || len(perr.Path) == 0 {
		return err.Error()
	}

	return fmt.Sprintf("%s: %s", FormatCtyPath(perr.Path), perr.Error())
}
// FormatCtyPath is a helper function to produce a user-friendly string
// representation of a cty.Path. The result uses a syntax similar to the
// HCL expression language in the hope of it being familiar to users.
func FormatCtyPath(path cty.Path) string {
	var buf bytes.Buffer
	for _, step := range path {
		switch ts := step.(type) {
		case cty.GetAttrStep:
			fmt.Fprintf(&buf, ".%s", ts.Name)
		case cty.IndexStep:
			buf.WriteByte('[')
			key := ts.Key
			keyTy := key.Type()
			switch {
			case key.IsNull():
				buf.WriteString("null")
			case !key.IsKnown():
				buf.WriteString("(not yet known)")
			case keyTy == cty.Number:
				bf := key.AsBigFloat()
				buf.WriteString(bf.Text('g', -1))
			case keyTy == cty.String:
				buf.WriteString(strconv.Quote(key.AsString()))
			default:
				buf.WriteString("...")
			}
			buf.WriteByte(']')
		}
	}
	return buf.String()
}
