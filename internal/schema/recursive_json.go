package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type jsonCoverEdge struct {
	schema  *document.Value
	pointer string // Original referenced root, including escaped segments.
}

type jsonCoverNode struct {
	index, low, height, weight int
	active                     bool
	edges                      []jsonCoverEdge
}

// Covers retain two source edges; transparent reference-only nodes retain one.
// Both must validate their original grammar before entering the graph.
func (a *analyzer) recursiveJSONEdges(v *document.Value, pointer string, depth int) (int, []jsonCoverEdge, *diagnostic.Diagnostic) {
	if v.Kind() != document.Object {
		return 0, nil, nil
	}
	var edges []jsonCoverEdge
	add := func(child *document.Value, p string, weight int) *diagnostic.Diagnostic {
		if child == nil || child.Kind() != document.Object || !has(child, "$ref") {
			return nil
		}
		if err := checkKeys(child, []string{"$ref"}, p); err != nil {
			return err
		}
		ref, valid := get(child, "$ref").Text()
		if !valid {
			return invalid("$ref must be a string.", naming.Pointer(p, "$ref"))
		}
		const prefix = "#/components/schemas/"
		if !strings.HasPrefix(ref, prefix) || strings.ContainsAny(ref[len(prefix):], "/%") {
			return nil
		}
		wire, err := naming.DecodePointerSegment(ref[len(prefix):], naming.Pointer(p, "$ref"))
		if err != nil {
			return err
		}
		target := get(get(get(a.root, "components"), "schemas"), wire)
		if target == nil {
			return nil // Preserve ordinary missing-reference diagnostics.
		}
		if err := depthCheck(depth+weight, p); err != nil {
			return err
		}
		edges = append(edges, jsonCoverEdge{target, naming.Pointer("/components/schemas", wire)})
		return nil
	}
	if has(v, "$ref") {
		if err := add(v, pointer, 1); err != nil {
			return 0, nil, err
		}
		if len(edges) != 1 {
			return 0, nil, nil
		}
		return 1, edges, nil
	}
	key := "oneOf"
	if !has(v, key) {
		key = "anyOf"
	}
	branches := get(v, key)
	if branches == nil || branches.Kind() != document.Array || len(branches.Items()) != 6 {
		return 0, nil, nil
	}
	seen := map[string]bool{}
	for _, branch := range branches.Items() {
		kind := text(get(branch, "type"))
		switch kind {
		case "null", "string", "boolean", "number", "array", "object":
		default:
			return 0, nil, nil
		}
		if seen[kind] {
			return 0, nil, nil
		}
		seen[kind] = true
	}
	if err := checkKeys(v, []string{key}, pointer); err != nil {
		return 0, nil, err
	}
	for i, branch := range branches.Items() {
		p := naming.Pointer(naming.Pointer(pointer, key), strconv.Itoa(i))
		if err := depthCheck(depth+1, p); err != nil {
			return 0, nil, err
		}
		allowed := []string{"type"}
		childKey := ""
		switch text(get(branch, "type")) {
		case "array":
			childKey = "items"
		case "object":
			childKey = "additionalProperties"
		}
		if childKey != "" {
			allowed = append(allowed, childKey)
		}
		if err := checkKeys(branch, allowed, p); err != nil {
			return 0, nil, err
		}
		if childKey != "" {
			if err := add(get(branch, childKey), naming.Pointer(p, childKey), 2); err != nil {
				return 0, nil, err
			}
		}
	}
	if len(edges) != 2 {
		return 0, nil, nil
	}
	return 2, edges, nil
}

// Strongly connected components close recursive JSON value edges without tree
// expansion. A group retains the sum of member source weights plus its longest
// outgoing group height. Only a complete cover can close a recursive group;
// transparent reference-only cycles never authorize a JSON value projection.
func (a *analyzer) recursiveJSONRoot(v *document.Value, pointer string) (int, *diagnostic.Diagnostic) {
	if !strings.HasPrefix(text(get(a.root, "openapi")), "3.1.") || has(v, "$ref") {
		return 0, nil
	}
	if height := a.recursiveJSONHeights[v]; height != 0 {
		return height, nil
	}
	nodes := map[*document.Value]*jsonCoverNode{}
	var stack []*jsonCoverNode
	work := 0
	var visit func(*document.Value, string, int) (*jsonCoverNode, *diagnostic.Diagnostic)
	visit = func(v *document.Value, pointer string, depth int) (*jsonCoverNode, *diagnostic.Diagnostic) {
		if node := nodes[v]; node != nil {
			return node, nil
		}
		work++
		if work > 16384 {
			return nil, diagnostic.At("E_SCHEMA_LIMIT", "Recursive JSON graph exceeds the supported work budget.", pointer)
		}
		weight, edges, err := a.recursiveJSONEdges(v, pointer, depth)
		if err != nil || weight == 0 {
			return nil, err
		}
		node := &jsonCoverNode{index: len(nodes), low: len(nodes), active: true, weight: weight, edges: edges}
		nodes[v] = node
		stack = append(stack, node)
		for _, edge := range edges {
			previous := nodes[edge.schema]
			next, err := visit(edge.schema, edge.pointer, depth+weight)
			if err != nil || next == nil {
				return nil, err
			}
			if next.active {
				if previous == nil {
					node.low = min(node.low, next.low)
				} else {
					node.low = min(node.low, next.index)
				}
			}
		}
		if node.low == node.index {
			var group []*jsonCoverNode
			for {
				n := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				n.active = false
				group = append(group, n)
				if n == node {
					break
				}
			}
			outgoing, weight, covers := 0, 0, 0
			for _, n := range group {
				weight += n.weight
				if n.weight == 2 {
					covers++
				}
				for _, edge := range n.edges {
					outgoing = max(outgoing, nodes[edge.schema].height)
				}
			}
			if covers == 0 && outgoing == 0 {
				return nil, nil
			}
			height := weight + outgoing
			if err := depthCheck(height, pointer); err != nil {
				return nil, err
			}
			for _, n := range group {
				n.height = height
			}
		}
		return node, nil
	}
	node, err := visit(v, pointer, 0)
	if err != nil || node == nil {
		return 0, err
	}
	if a.recursiveJSONHeights == nil {
		a.recursiveJSONHeights = map[*document.Value]int{}
	}
	for original, n := range nodes {
		if n.weight == 2 {
			a.recursiveJSONHeights[original] = n.height
		}
	}
	return node.height, nil
}
