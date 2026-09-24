// Package validate kiểm một bảng đã parse theo lược đồ metadata và theo các
// luật riêng của sơ đồ activity-swimlane.
package validate

import (
	"fmt"
	"github.com/luytbq/flowcast/internal/unistr"
	"strings"

	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/schema"
)

// Thông điệp dùng dấu nháy viết thẳng chứ không dùng %q. %q thoát ký tự đặc
// biệt, nên một key metadata có gạch chéo ngược sẽ in ra khác với cái người
// dùng đã viết trong bảng.

// Validate trả về mọi phát hiện, theo thứ tự cố định.
//
// Thứ tự là một phần của giao diện: caller in chúng ra theo đúng thứ tự này, và
// cổng đối chiếu so cả thứ tự lẫn nội dung.
func Validate(rows []model.Row) []model.Issue {
	v := &validator{rows: rows, byID: map[string]int{}}
	v.checkIDs()
	v.checkRows()
	v.checkGraph()
	return v.issues
}

type validator struct {
	rows   []model.Row
	issues []model.Issue
	// byID giữ chỉ số của dòng đầu tiên mang id đó. Dòng trùng id sau đó bị bỏ
	// qua ở các kiểm tra về đồ thị.
	byID  map[string]int
	lanes map[string]bool
}

func (v *validator) err(r model.Row, code, msg string) {
	v.issues = append(v.issues, model.Issue{Code: code, Level: model.LevelError, Loc: r.Loc, ID: r.ID, Msg: msg})
}

func (v *validator) warn(r model.Row, code, msg string) {
	v.issues = append(v.issues, model.Issue{Code: code, Level: model.LevelWarning, Loc: r.Loc, ID: r.ID, Msg: msg})
}

func (v *validator) row(id string) (model.Row, bool) {
	i, ok := v.byID[id]
	if !ok {
		return model.Row{}, false
	}
	return v.rows[i], true
}

func (v *validator) isNode(id string) bool {
	r, ok := v.row(id)
	return ok && schema.NodeTypes[r.Type]
}

func (v *validator) checkIDs() {
	for i, r := range v.rows {
		if r.ID == "" {
			v.err(r, "table.empty_id", "id trống")
			continue
		}
		if schema.ReservedIDs[r.ID] {
			v.err(r, "table.reserved_id", fmt.Sprintf("id \"%s\" trùng id dành riêng của draw.io", r.ID))
		}
		if j, dup := v.byID[r.ID]; dup {
			v.err(r, "ref.duplicate_id", "id trùng với "+v.rows[j].Loc.String())
		} else {
			v.byID[r.ID] = i
		}
		if !schema.AllTypes[r.Type] {
			v.err(r, "table.bad_type", fmt.Sprintf("type không hợp lệ: \"%s\"", r.Type))
		}
	}
	v.lanes = map[string]bool{}
	for _, r := range v.rows {
		if r.Type == "lane" {
			v.lanes[r.ID] = true
		}
	}
}

// isMarker nhận dòng text đánh dấu phần còn lại của bảng. Dòng đó không phải
// một phần tử nên không cần parent.
func isMarker(r model.Row) bool {
	return r.Type == "text" && r.Text() == schema.RestMarker && r.Meta["attach"] == ""
}

func (v *validator) checkRows() {
	for _, r := range v.rows {
		if !schema.AllTypes[r.Type] {
			continue
		}
		switch {
		case r.Type == "lane" || r.Type == "edge":
			if r.Parent != "" {
				v.err(r, "table.parent_not_empty", fmt.Sprintf("%s phải để trống parent", r.Type))
			}
		case !isMarker(r):
			// Bảng không có lane là flowchart: mọi phần tử nằm chung một vùng
			// và để trống parent.
			if r.Parent == "" && len(v.lanes) == 0 {
				break
			}
			if r.Parent == "" {
				v.err(r, "ref.missing_parent", "thiếu parent (id của lane chứa phần tử)")
			} else if !v.lanes[r.Parent] {
				v.err(r, "ref.parent_not_lane", fmt.Sprintf("parent \"%s\" không phải id của một lane", r.Parent))
			}
		}

		spec := schema.For(r.Type)
		for _, k := range r.MetaKeys {
			if !spec.AllowsKey(k) {
				v.warn(r, "schema.unknown_key",
					fmt.Sprintf("metadata \"%s\" không dùng cho type %s, bị bỏ qua", k, r.Type))
			}
		}
		for _, s := range styleList(r) {
			if !spec.AllowsStyle(s) {
				v.warn(r, "schema.bad_style",
					fmt.Sprintf("style \"%s\" không dùng cho type %s, bị bỏ qua", s, r.Type))
			}
		}
		if b, ok := r.Meta["back"]; ok && b != "true" && b != "false" {
			v.warn(r, "schema.bad_back", fmt.Sprintf("back=\"%s\" không hợp lệ, chỉ nhận true/false", b))
		}
		v.checkRefs(r, spec)
	}
}

func (v *validator) checkRefs(r model.Row, spec schema.TypeSpec) {
	for _, ref := range spec.Refs {
		val := r.Meta[ref.Key]
		if val == "" {
			if ref.Required {
				v.err(r, "schema.missing_key", fmt.Sprintf("%s thiếu %s", r.Type, ref.Key))
			}
			continue
		}
		target, ok := v.row(val)
		if !ok {
			v.err(r, "ref.dangling", fmt.Sprintf("%s=%s trỏ tới id không tồn tại", ref.Key, val))
			continue
		}
		if !schema.NodeTypes[target.Type] {
			v.err(r, "ref.bad_target",
				fmt.Sprintf("%s=%s là %s; %s", ref.Key, val, target.Type, ref.Note))
			continue
		}
		if ref.SameLane && target.Parent != r.Parent {
			v.warn(r, "ref.other_lane",
				fmt.Sprintf("nằm ở lane %s nhưng gắn vào %s thuộc lane %s; sẽ vẽ trong lane %s",
					r.Parent, val, target.Parent, target.Parent))
		}
	}
}

// styleList trả về giá trị style theo đúng thứ tự viết trong ô, đã bỏ trùng.
func styleList(r model.Row) []string {
	var out []string
	for _, s := range strings.Split(r.Meta["style"], ",") {
		s = unistr.Strip(s)
		if s == "" {
			continue
		}
		dup := false
		for _, x := range out {
			if x == s {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, s)
		}
	}
	return out
}

func (v *validator) checkGraph() {
	var edges []model.Row
	outs := map[string][]model.Row{}
	ins := map[string][]model.Row{}
	for _, r := range v.rows {
		if r.Type != "edge" || !v.isNode(r.Meta["from"]) || !v.isNode(r.Meta["to"]) {
			continue
		}
		edges = append(edges, r)
		outs[r.Meta["from"]] = append(outs[r.Meta["from"]], r)
		ins[r.Meta["to"]] = append(ins[r.Meta["to"]], r)
	}
	// Id trùng thì vị trí lấy theo dòng cuối cùng mang id đó.
	pos := map[string]int{}
	for i, r := range v.rows {
		if r.ID != "" {
			pos[r.ID] = i
		}
	}

	for i, r := range v.rows {
		if !schema.NodeTypes[r.Type] {
			continue
		}
		if j, ok := v.byID[r.ID]; !ok || j != i {
			continue
		}
		es := outs[r.ID]
		if r.Type == "condition" && len(es) < 2 {
			v.err(r, "graph.condition_branches",
				fmt.Sprintf("condition chỉ có %d cạnh ra, cần ít nhất 2", len(es)))
		}
		if (r.Type == "task" || r.Type == "condition") && len(es) == 0 {
			v.warn(r, "graph.no_out_edge", fmt.Sprintf("%s không có cạnh ra", r.Type))
		}
		if r.Type == "start" && len(ins[r.ID]) > 0 {
			v.warn(r, "graph.start_has_in", "start có cạnh đi vào: "+ids(ins[r.ID]))
		}
		if r.Type == "end" && len(es) > 0 {
			v.warn(r, "graph.end_has_out", "end có cạnh đi ra: "+ids(es))
		}
		if r.Type == "condition" {
			for _, e := range es {
				if e.Text() == "" {
					v.warn(e, "graph.branch_no_label",
						fmt.Sprintf("cạnh ra của condition %s không có nhãn", r.ID))
				}
			}
		}
		v.checkOutEdgeOrder(r, es, pos)
	}

	for _, e := range edges {
		if e.Meta["back"] == "true" {
			continue
		}
		if pos[e.Meta["to"]] < pos[e.Meta["from"]] {
			v.warn(e, "order.missing_back",
				fmt.Sprintf("%s đứng trước nguồn %s nhưng cạnh không ghi back=true",
					e.Meta["to"], e.Meta["from"]))
		}
	}
}

// checkOutEdgeOrder canh luật đọc của định dạng: cạnh ra của một phần tử phải
// nằm liền ngay sau phần tử đó, chỉ được chen các db và text gắn vào chính nó.
func (v *validator) checkOutEdgeOrder(r model.Row, es []model.Row, pos map[string]int) {
	if len(es) == 0 {
		return
	}
	p := pos[r.ID] + 1
	for p < len(v.rows) && schema.AttachTypes[v.rows[p].Type] && v.rows[p].Meta["attach"] == r.ID {
		p++
	}
	block := map[string]bool{}
	for k := p; k < len(v.rows) && k < p+len(es); k++ {
		block[v.rows[k].ID] = true
	}
	var misplaced []string
	for _, e := range es {
		if !block[e.ID] {
			misplaced = append(misplaced, e.ID)
		}
	}
	if len(misplaced) == 0 && len(block) == len(es) {
		return
	}
	v.err(r, "order.out_edges_not_adjacent",
		fmt.Sprintf("cạnh ra phải nằm liền sau %s (và các db/text gắn vào nó); đang lệch: %s",
			r.ID, strings.Join(misplaced, ", ")))
}

func ids(rows []model.Row) string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return strings.Join(out, ", ")
}
