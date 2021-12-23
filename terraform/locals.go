package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// Local represents a single entry from a "locals" block in a module or file.
// The "locals" block itself is not represented, because it serves only to
// provide context for us to interpret its contents.
type Local struct {
	Name      string
	Expr      hcl.Expression
	Value     cty.Value
	DeclRange hcl.Range
}

func decodeLocalsBlock(block *hcl.Block) ([]Local, hcl.Diagnostics) {
	attrs, diags := block.Body.JustAttributes()
	if len(attrs) == 0 {
		return nil, diags
	}

	locals := make([]Local, 0, len(attrs))
	for name, attr := range attrs {
		local := Local{
			Name:      name,
			Expr:      attr.Expr,
			DeclRange: attr.Range,
		}
		val, valDiags := local.Expr.Value(nil)
		diags = append(diags, valDiags...)
		local.Value = val
		locals = append(locals, local)
	}
	return locals, diags
}
