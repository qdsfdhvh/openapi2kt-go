package config

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
	"openapi-kt/internal/diagnostic"
)

// The AST API is isolated here and pinned in go.mod. Upgrades must rerun the
// grammar, duplicate-location and budget regressions before changing the pin.
// The library bounds its own initial value parsing to 10,000 nested containers;
// this iterative preflight imposes our smaller limits before map decoding and
// the library's duplicate tracker (which searches prior keys linearly).
const (
	MaxDepth = 64
	MaxNodes = 65536
	MaxWork  = 32 * 1024 * 1024
)

type sourceLocation struct {
	offset  int
	pointer string
}
type sourceIndex struct {
	order       map[string]int
	expressions []sourceLocation
}

type pathPart struct {
	key   string
	index bool // true only for adapter-created array positions, never TOML keys
}
type configPath []pathPart

func keyPath(keys ...string) configPath {
	return extendPath(nil, keys)
}
func arrayPath(base configPath, index int) configPath {
	out := append(configPath{}, base...)
	return append(out, pathPart{key: strconv.Itoa(index), index: true})
}

func pathKey(parts configPath) string {
	var b strings.Builder
	for _, part := range parts {
		b.WriteByte('/')
		b.WriteString(strings.ReplaceAll(strings.ReplaceAll(part.key, "~", "~0"), "/", "~1"))
	}
	return b.String()
}

// Unknown input keys are never printed, even as JSON Pointer segments. Retain
// the closest known schema container or field, including the spec index.
func safePointer(parts configPath) string {
	if len(parts) == 0 {
		return ""
	}
	n := 1
	switch parts[0].key {
	case "config_version":
	case "defaults":
		if len(parts) > 1 && knownSetting(parts[1].key) {
			n++
		}
	case "specs":
		if len(parts) > 1 && parts[1].index {
			n++
			if len(parts) > 2 && knownSetting(parts[2].key) {
				n++
			}
		}
	default:
		if !knownSetting(parts[0].key) {
			return ""
		}
	}
	return pathKey(parts[:n])
}

func located(code, pointer string) *diagnostic.Diagnostic {
	d := problem(code)
	d.Pointer = &pointer
	return d
}

func nodeKey(n *unstable.Node) []string {
	var key []string
	it := n.Key()
	for it.Next() {
		key = append(key, string(it.Node().Data))
	}
	return key
}

func extendPath(base configPath, key []string) configPath {
	p := make(configPath, 0, len(base)+len(key))
	p = append(p, base...)
	for _, part := range key {
		p = append(p, pathPart{key: part})
	}
	return p
}

func inspectTOML(data []byte) (sourceIndex, *diagnostic.Diagnostic) {
	source := sourceIndex{order: map[string]int{}}
	var parser unstable.Parser
	parser.Reset(data)
	var scope configPath
	specIndex := -1
	nodes, keys, anonymous, tables, lastEnd := 0, 0, 0, 0, 0
	// The pinned tracker creates anonymous entries for containers in arrays
	// and scans them in later key lookups. Charge all potential anonymous
	// entries against every key, regardless of their traversal/source order.
	// Array-table clears also scan/remap the entry inventory.
	work := func() int64 {
		k, a, t := int64(keys), int64(anonymous), int64(tables)
		return int64(nodes) + k*(k+1)/2 + k*a + 3*t*(k+a+1)
	}
	for parser.NextExpression() {
		expr := parser.Expression()
		key := nodeKey(expr)
		parts := extendPath(scope, key)
		if expr.Kind == unstable.Table || expr.Kind == unstable.ArrayTable {
			parts = keyPath(key...)
			if expr.Kind == unstable.ArrayTable && len(key) == 1 && key[0] == "specs" {
				specIndex++
				if specIndex >= MaxSpecs {
					return source, located("E_CONFIG_LIMIT", "/specs")
				}
				parts = arrayPath(parts, specIndex)
			} else if len(key) > 1 && key[0] == "specs" && specIndex >= 0 {
				parts = extendPath(arrayPath(keyPath("specs"), specIndex), key[1:])
			}
			scope = parts
		}
		firstKey := expr.Key()
		firstKey.Next()
		offset := int(firstKey.Node().Raw.Offset)
		pointer := safePointer(parts)
		source.expressions = append(source.expressions, sourceLocation{offset, pointer})
		// Record all keys/values, including inline tables, without recursive
		// walks. Their source positions order schema diagnostics deterministically.
		type frame struct {
			node  *unstable.Node
			path  configPath
			depth int
		}
		stack := []frame{{expr, scope, 0}}
		if expr.Kind == unstable.KeyValue {
			stack[0].path = parts[:len(parts)-len(key)]
		}
		for len(stack) > 0 {
			f := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			nodes++
			if nodes > MaxNodes || work() > MaxWork || f.depth > MaxDepth || len(f.path) > MaxDepth {
				return source, located("E_CONFIG_LIMIT", safePointer(f.path))
			}
			n := f.node
			end := int(n.Raw.Offset + n.Raw.Length)
			if end > lastEnd {
				lastEnd = end
			}
			switch n.Kind {
			case unstable.KeyValue, unstable.Table, unstable.ArrayTable:
				key := nodeKey(n)
				// Charge a conservative upper bound on duplicate-tracker key
				// comparisons, in addition to AST visits. Account for all key
				// parts, not just top-level assignments.
				if n.Kind == unstable.ArrayTable {
					tables++
				}
				for range key {
					keys++
					nodes++
				}
				p := f.path
				if n.Kind == unstable.KeyValue {
					p = extendPath(p, key)
				}
				if nodes > MaxNodes || work() > MaxWork || len(p) > MaxDepth {
					return source, located("E_CONFIG_LIMIT", safePointer(p))
				}
				it := n.Key()
				it.Next()
				for i := 1; i <= len(p); i++ {
					if _, exists := source.order[pathKey(p[:i])]; !exists {
						source.order[pathKey(p[:i])] = int(it.Node().Raw.Offset)
					}
				}
				if n.Kind == unstable.KeyValue {
					stack = append(stack, frame{n.Value(), p, f.depth})
				}
			case unstable.Array:
				i := 0
				it := n.Children()
				for it.Next() {
					element := it.Node()
					if element.Kind == unstable.Array || element.Kind == unstable.InlineTable {
						anonymous++
					}
					stack = append(stack, frame{element, arrayPath(f.path, i), f.depth + 1})
					i++
				}
				if work() > MaxWork {
					return source, located("E_CONFIG_LIMIT", safePointer(f.path))
				}
			case unstable.InlineTable:
				it := n.Children()
				for it.Next() {
					stack = append(stack, frame{it.Node(), f.path, f.depth + 1})
				}
			}
		}
		if expr.Kind == unstable.Table || expr.Kind == unstable.ArrayTable {
			it := expr.Key()
			for it.Next() {
				end := int(it.Node().Raw.Offset + it.Node().Raw.Length)
				if end > lastEnd {
					lastEnd = end
				}
			}
			// Header Raw ranges cover keys, so skip the remaining delimiters
			// and comment before locating a subsequent malformed assignment.
			if end := strings.IndexByte(string(data[lastEnd:]), '\n'); end >= 0 {
				lastEnd += end + 1
			} else {
				lastEnd = len(data)
			}
		}
	}
	if err := parser.Error(); err != nil {
		// No parser message, raw key, value or contextual excerpt crosses the
		// adapter. The library's nesting guard can fire during initial parsing.
		var pe *unstable.ParserError
		if errors.As(err, &pe) && pe.Message == "arrays and inline tables are nested more than the maximum of 10000 levels deep" {
			return source, located("E_CONFIG_LIMIT", safePointer(scope))
		}
		return source, located("E_CONFIG_SYNTAX", syntaxPointer(data[lastEnd:], scope))
	}
	return source, nil
}

// Recover only the pending assignment's key using the library itself. This
// fallback is for syntax errors before a complete AST expression exists; it
// returns a known ancestor when the key/header cannot be parsed safely.
func syntaxPointer(pending []byte, scope configPath) string {
	if equal := strings.IndexByte(string(pending), '='); equal >= 0 {
		prefix := append([]byte{}, pending[:equal+1]...)
		prefix = append(prefix, '0')
		var p unstable.Parser
		p.Reset(prefix)
		if p.NextExpression() && p.Expression().Kind == unstable.KeyValue {
			return safePointer(extendPath(scope, nodeKey(p.Expression())))
		}
	}
	return safePointer(scope)
}

func (s sourceIndex) sortedKeys(table map[string]any, scope configPath) []string {
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a := s.order[pathKey(extendPath(scope, []string{keys[i]}))]
		b := s.order[pathKey(extendPath(scope, []string{keys[j]}))]
		if a == b {
			return keys[i] < keys[j]
		}
		return a < b
	})
	return keys
}

func decodeTOML(data []byte) (map[string]any, sourceIndex, *diagnostic.Diagnostic) {
	source, failure := inspectTOML(data)
	if failure != nil {
		return nil, source, failure
	}
	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		pointer := ""
		var de *toml.DecodeError
		if errors.As(err, &de) {
			row, col := de.Position()
			offset := 0
			for i, c := range data {
				if row == 1 {
					break
				}
				if c == '\n' {
					row--
					offset = i + 1
				}
			}
			offset += col - 1
			for _, loc := range source.expressions {
				if loc.offset > offset {
					break
				}
				pointer = loc.pointer
			}
		}
		// A syntactically complete AST may still contain malformed numeric/
		// date tokens (validated on decoding) or duplicate declarations. The
		// pinned library uses these phrases for its declaration conflicts.
		code := "E_CONFIG_SYNTAX"
		if errors.As(err, &de) && (strings.Contains(de.Error(), "already exists") || strings.Contains(de.Error(), "already defined") || strings.Contains(de.Error(), "should be a table, not")) {
			code = "E_CONFIG_DUPLICATE"
			if len(de.Key()) == 1 && de.Key()[0] == "specs" {
				pointer = "/specs"
			}
		}
		return nil, source, located(code, pointer)
	}
	return root, source, nil
}
