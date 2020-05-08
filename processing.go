package main

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
)

type tfsources struct {
	resources map[string][]*hcl.Block
	data      map[string][]*hcl.Block
	modules   map[string][]*hcl.Block

	ctx *hcl.EvalContext
}

func (t tfsources) debug() (debugStr string) {
	ri := 0
	for typ, resource := range t.resources {
		if ri == 0 {
			debugStr += "resources:\n"
		}
		for i, v := range resource {
			debugStr += fmt.Sprintf(
				"[%d/%d] resource:%q [%d/%d] %q\n",
				ri,
				len(t.resources),
				typ,
				i,
				len(resource),
				v.Labels[1],
			)
		}
		ri++
	}

	di := 0
	for typ, data := range t.data {
		if di == 0 {
			debugStr += "\ndata:\n"
		}
		for i, v := range data {
			debugStr += fmt.Sprintf(
				"[%d/%d] data:%q [%d/%d] %q\n",
				di,
				len(t.data),
				typ,
				i,
				len(data),
				v.Labels[1],
			)
		}
		di++
	}

	mi := 0
	for name, module := range t.modules {
		if mi == 0 {
			debugStr += "\nmodules:\n"
		}
		for _, v := range module {
			attrs, _ := v.Body.JustAttributes()
			var source string
			for _, attr := range attrs {
				if attr.Name == "source" {
					sourceVal, _ := attr.Expr.Value(t.ctx)
					if sourceVal.Type() == cty.String {
						source = sourceVal.AsString()
					}
					break
				}
			}
			debugStr += fmt.Sprintf(
				"[%d/%d] module:%q %q\n",
				mi,
				len(t.modules),
				name,
				source,
			)
		}
		mi++
	}

	return
}

func (t tfsources) processFile(file *hcl.File) {
	contents, _ := file.Body.Content(terraformSchema)

	for _, block := range contents.Blocks {
		var aggr map[string][]*hcl.Block

		switch block.Type {
		case "resource":
			aggr = t.resources
			fallthrough
		case "data":
			if aggr == nil {
				aggr = t.data
			}

			if len(block.Labels) < 2 {
				// broken resource, toss it
				continue
			}

			typ := block.Labels[0]
			aggr[typ] = append(aggr[typ], block)

		case "module":
			t.modules[block.Labels[0]] = []*hcl.Block{block}

		default:
			continue
		}
	}
}
