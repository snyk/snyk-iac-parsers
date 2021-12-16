package terraform

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Resource represents a "resource" or "data" block in a module or file.
type Resource struct {
	//Mode    addrs.ResourceMode
	Name    string
	Type    string
	Config  hcl.Body
	Count   hcl.Expression
	ForEach hcl.Expression

	//ProviderConfigRef *ProviderConfigRef
	//Provider          addrs.Provider

	DependsOn []hcl.Traversal

	// Managed is populated only for Mode = addrs.ManagedResourceMode,
	// containing the additional fields that apply to managed resources.
	// For all other resource modes, this field is nil.
	//Managed *ManagedResource

	DeclRange hcl.Range
	TypeRange hcl.Range
	Block *hcl.Block
}


var resourceBlockSchema = &hcl.BodySchema{
	Attributes: commonResourceAttributes,
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "locals"}, // reserved for future use
		{Type: "lifecycle"},
		{Type: "connection"},
		{Type: "provisioner", LabelNames: []string{"type"}},
		{Type: "_"}, // meta-argument escaping block
	},
}

var resourceLifecycleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "create_before_destroy",
		},
		{
			Name: "prevent_destroy",
		},
		{
			Name: "ignore_changes",
		},
	},
}

var commonResourceAttributes = []hcl.AttributeSchema{
	{
		Name: "count",
	},
	{
		Name: "for_each",
	},
	{
		Name: "provider",
	},
	{
		Name: "depends_on",
	},
}

func decodeResourceBlock(block *hcl.Block) (*Resource, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	r := &Resource{
		//Mode:      addrs.ManagedResourceMode,
		Type:      block.Labels[0],
		Name:      block.Labels[1],
		DeclRange: block.DefRange,
		TypeRange: block.LabelRanges[0],
		//Managed:   &ManagedResource{},
		Block: block,
	}

	content, remain, moreDiags := block.Body.PartialContent(resourceBlockSchema)
	diags = append(diags, moreDiags...)
	r.Config = remain

	if !hclsyntax.ValidIdentifier(r.Type) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid resource type name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}
	if !hclsyntax.ValidIdentifier(r.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid resource name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[1],
		})
	}

	if attr, exists := content.Attributes["count"]; exists {
		r.Count = attr.Expr
	}

	if attr, exists := content.Attributes["for_each"]; exists {
		r.ForEach = attr.Expr
		// Cannot have count and for_each on the same resource block
		if r.Count != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid combination of "count" and "for_each"`,
				Detail:   `The "count" and "for_each" meta-arguments are mutually-exclusive, only one should be used to be explicit about the number of resources to be created.`,
				Subject:  &attr.NameRange,
			})
		}
	}
	//
	//if attr, exists := content.Attributes["provider"]; exists {
	//	var providerDiags hcl.Diagnostics
	//	r.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr.Expr, "provider")
	//	diags = append(diags, providerDiags...)
	//}

	if attr, exists := content.Attributes["depends_on"]; exists {
		deps, depsDiags := decodeDependsOn(attr)
		diags = append(diags, depsDiags...)
		r.DependsOn = append(r.DependsOn, deps...)
	}

	var seenLifecycle *hcl.Block
	//var seenConnection *hcl.Block
	var seenEscapeBlock *hcl.Block
	for _, block := range content.Blocks {
		switch block.Type {
		case "lifecycle":
			if seenLifecycle != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate lifecycle block",
					Detail:   fmt.Sprintf("This resource already has a lifecycle block at %s.", seenLifecycle.DefRange),
					Subject:  &block.DefRange,
				})
				continue
			}
			seenLifecycle = block

			_, lcDiags := block.Body.Content(resourceLifecycleBlockSchema)
			diags = append(diags, lcDiags...)

			//if attr, exists := lcContent.Attributes["create_before_destroy"]; exists {
			//	valDiags := gohcl.DecodeExpression(attr.Expr, nil, &r.Managed.CreateBeforeDestroy)
			//	diags = append(diags, valDiags...)
			//	r.Managed.CreateBeforeDestroySet = true
			//}
			//
			//if attr, exists := lcContent.Attributes["prevent_destroy"]; exists {
			//	valDiags := gohcl.DecodeExpression(attr.Expr, nil, &r.Managed.PreventDestroy)
			//	diags = append(diags, valDiags...)
			//	r.Managed.PreventDestroySet = true
			//}

			//if attr, exists := lcContent.Attributes["ignore_changes"]; exists {
			//
			//	// ignore_changes can either be a list of relative traversals
			//	// or it can be just the keyword "all" to ignore changes to this
			//	// resource entirely.
			//	//   ignore_changes = [ami, instance_type]
			//	//   ignore_changes = all
			//	// We also allow two legacy forms for compatibility with earlier
			//	// versions:
			//	//   ignore_changes = ["ami", "instance_type"]
			//	//   ignore_changes = ["*"]
			//
			//	kw := hcl.ExprAsKeyword(attr.Expr)
			//
			//	switch {
			//	case kw == "all":
			//		r.Managed.IgnoreAllChanges = true
			//	default:
			//		exprs, listDiags := hcl.ExprList(attr.Expr)
			//		diags = append(diags, listDiags...)
			//
			//		var ignoreAllRange hcl.Range
			//
			//		for _, expr := range exprs {
			//
			//			// our expr might be the literal string "*", which
			//			// we accept as a deprecated way of saying "all".
			//			if shimIsIgnoreChangesStar(expr) {
			//				r.Managed.IgnoreAllChanges = true
			//				ignoreAllRange = expr.Range()
			//				diags = append(diags, &hcl.Diagnostic{
			//					Severity: hcl.DiagError,
			//					Summary:  "Invalid ignore_changes wildcard",
			//					Detail:   "The [\"*\"] form of ignore_changes wildcard is was deprecated and is now invalid. Use \"ignore_changes = all\" to ignore changes to all attributes.",
			//					Subject:  attr.Expr.Range().Ptr(),
			//				})
			//				continue
			//			}
			//
			//			expr, shimDiags := shimTraversalInString(expr, false)
			//			diags = append(diags, shimDiags...)
			//
			//			traversal, travDiags := hcl.RelTraversalForExpr(expr)
			//			diags = append(diags, travDiags...)
			//			if len(traversal) != 0 {
			//				r.Managed.IgnoreChanges = append(r.Managed.IgnoreChanges, traversal)
			//			}
			//		}
			//
			//		if r.Managed.IgnoreAllChanges && len(r.Managed.IgnoreChanges) != 0 {
			//			diags = append(diags, &hcl.Diagnostic{
			//				Severity: hcl.DiagError,
			//				Summary:  "Invalid ignore_changes ruleset",
			//				Detail:   "Cannot mix wildcard string \"*\" with non-wildcard references.",
			//				Subject:  &ignoreAllRange,
			//				Context:  attr.Expr.Range().Ptr(),
			//			})
			//		}
			//
			//	}
			//
			//}

		//case "connection":
		//	if seenConnection != nil {
		//		diags = append(diags, &hcl.Diagnostic{
		//			Severity: hcl.DiagError,
		//			Summary:  "Duplicate connection block",
		//			Detail:   fmt.Sprintf("This resource already has a connection block at %s.", seenConnection.DefRange),
		//			Subject:  &block.DefRange,
		//		})
		//		continue
		//	}
		//	seenConnection = block
		//
		//	r.Managed.Connection = &Connection{
		//		Config:    block.Body,
		//		DeclRange: block.DefRange,
		//	}

		//case "provisioner":
		//	pv, pvDiags := decodeProvisionerBlock(block)
		//	diags = append(diags, pvDiags...)
		//	if pv != nil {
		//		r.Managed.Provisioners = append(r.Managed.Provisioners, pv)
		//	}

		case "_":
			if seenEscapeBlock != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate escaping block",
					Detail: fmt.Sprintf(
						"The special block type \"_\" can be used to force particular arguments to be interpreted as resource-type-specific rather than as meta-arguments, but each resource block can have only one such block. The first escaping block was at %s.",
						seenEscapeBlock.DefRange,
					),
					Subject: &block.DefRange,
				})
				continue
			}
			seenEscapeBlock = block

			// When there's an escaping block its content merges with the
			// existing config we extracted earlier, so later decoding
			// will see a blend of both.
			r.Config = hcl.MergeBodies([]hcl.Body{r.Config, block.Body})

		default:
			// Any other block types are ones we've reserved for future use,
			// so they get a generic message.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reserved block type name in resource block",
				Detail:   fmt.Sprintf("The block type name %q is reserved for use by Terraform in a future version.", block.Type),
				Subject:  &block.TypeRange,
			})
		}
	}

	// Now we can validate the connection block references if there are any destroy provisioners.
	// TODO: should we eliminate standalone connection blocks?
	//if r.Managed.Connection != nil {
	//	for _, p := range r.Managed.Provisioners {
	//		if p.When == ProvisionerWhenDestroy {
	//			diags = append(diags, onlySelfRefs(r.Managed.Connection.Config)...)
	//			break
	//		}
	//	}
	//}

	return r, diags
}