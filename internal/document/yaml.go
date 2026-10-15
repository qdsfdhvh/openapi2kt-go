package document

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"

	"openapi-kt/internal/diagnostic"
)

// YAML-specific work budgets do not change the existing JSON/schema limits.
// Count source nodes (including keys) and expanded visits separately. Aliases
// consume a visit before their target, so even chains of scalar aliases are bounded.
const maxYAMLNodes = 100_000
const maxYAMLAliasDepth = 127

var jsonNumber = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)
var jsonInteger = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

// The upstream resolver falls back to !!str on numeric overflow. Recognize
// numeric spellings independently (including its legacy 0b+1 / 0o-1 forms), so
// forbidden forms cannot become strings or string keys by exceeding its range.
var yamlNumberLike = regexp.MustCompile(`^([+-]?(0[xX][0-9a-fA-F]+|0[oO][0-7]+|0[bB][01]+|[0-9]+(\.[0-9]*)?([eE][+-]?[0-9]+)?|\.[0-9]+([eE][+-]?[0-9]+)?)|0b[+-][01]+|0o[+-][0-7]+)$`)

func plainYAMLNumber(n *yaml.Node) bool {
	if n.Kind != yaml.ScalarNode || n.Style != 0 || len(n.Value) == 0 {
		return false
	}
	first := n.Value[0]
	if (first < '0' || first > '9') && first != '+' && first != '-' && first != '.' {
		return false
	}
	// Match the upstream resolver's underscore removal before numeric
	// recognition. Require a complete numeric spelling afterwards: 0x_ and
	// 1e_ remain strings, while 0_xFF and +_1e400 cannot evade rejection.
	return yamlNumberLike.MatchString(strings.ReplaceAll(n.Value, "_", ""))
}

func yamlError(code, message string, n *yaml.Node) *diagnostic.Diagnostic {
	if n != nil && n.Line > 0 && n.Column > 0 {
		message = fmt.Sprintf("%s At line %d, column %d.", message, n.Line, n.Column)
	}
	return diagnostic.New(code, message)
}

func parseYAML(data []byte) (*Value, *diagnostic.Diagnostic) {
	if !utf8.Valid(data) {
		return nil, yamlError("E_YAML_INVALID", "YAML input must be valid UTF-8.", nil)
	}
	if hasYAMLTagDirective(data) {
		return nil, yamlError("E_YAML_UNSUPPORTED", "YAML tag directives are unsupported.", nil)
	}
	// Node decoding preserves order/tokens and never invokes Go typed decoding,
	// merge resolution, time conversion or the library's alias expansion.
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := decoder.Decode(&doc); err != nil {
		return nil, yamlError("E_YAML_INVALID", "Invalid or empty YAML document.", nil)
	}
	if len(doc.Content) != 1 {
		return nil, yamlError("E_YAML_INVALID", "Expected one YAML document.", nil)
	}
	var next yaml.Node
	if err := decoder.Decode(&next); err != io.EOF {
		return nil, yamlError("E_YAML_INVALID", "Multiple documents or trailing YAML are unsupported.", nil)
	}
	state := yamlTree{anchors: make(map[string]*yaml.Node), active: make(map[*yaml.Node]bool)}
	if err := state.checkSource(doc.Content[0], 0); err != nil {
		return nil, err
	}
	if err := checkYAMLTagIdentity(data, doc.Content[0]); err != nil {
		return nil, err
	}
	value, err := state.expand(doc.Content[0], 0, 0)
	if err != nil {
		return nil, err
	}
	if value.Kind() != Object {
		return nil, diagnostic.At("E_DOCUMENT_INVALID", "The document root must be an object.", "")
	}
	return value, nil
}

// The upstream parser concatenates directive URI prefixes onto each tagged
// node before returning its Node tree. Reject %TAG lines before that allocation
// amplification, even in quoted continuations. This is a conservative source
// guard, not a tokenizer: ordinary syntax/strings still use the YAML parser.
// All upstream YAML line breaks and a column-zero BOM are covered.
func hasYAMLTagDirective(data []byte) bool {
	start := true
	for i := 0; i < len(data); {
		r, width := utf8.DecodeRune(data[i:])
		if start && r == '%' && bytes.HasPrefix(data[i:], []byte("%TAG")) {
			return true
		}
		if !(start && r == '\ufeff') {
			start = r == '\n' || r == '\r' || r == '\u0085' || r == '\u2028' || r == '\u2029'
		}
		i += width
	}
	return false
}

type yamlTree struct {
	anchors map[string]*yaml.Node
	active  map[*yaml.Node]bool
	source  int
	visits  int
	bytes   int
}

// Only Content is traversed here: Alias points back into the source graph.
// Validate unused anchors too, and reject rebinding rather than silently using
// the library's last-anchor-wins interpretation.
func (s *yamlTree) checkSource(n *yaml.Node, depth int) *diagnostic.Diagnostic {
	s.source++
	if s.source > maxYAMLNodes {
		return yamlError("E_YAML_LIMIT", "YAML source node budget exceeded.", n)
	}
	if n.Kind == yaml.MappingNode || n.Kind == yaml.SequenceNode {
		if depth >= MaxJSONDepth {
			return yamlError("E_YAML_LIMIT", "YAML container depth exceeds 127.", n)
		}
		depth++
	}
	if n.Anchor != "" {
		if _, exists := s.anchors[n.Anchor]; exists {
			return yamlError("E_YAML_INVALID", "Duplicate YAML anchor declaration.", n)
		}
		s.anchors[n.Anchor] = n
	}
	for _, child := range n.Content {
		if err := s.checkSource(child, depth); err != nil {
			return err
		}
	}
	return nil
}

func (s *yamlTree) addBytes(count int, n *yaml.Node) *diagnostic.Diagnostic {
	if count > MaxDocumentBytes-s.bytes {
		return yamlError("E_YAML_LIMIT", "Expanded JSON tree exceeds the 8 MiB size budget.", n)
	}
	s.bytes += count
	return nil
}

func (s *yamlTree) expand(n *yaml.Node, depth, aliases int) (*Value, *diagnostic.Diagnostic) {
	s.visits++
	if s.visits > maxYAMLNodes {
		return nil, yamlError("E_YAML_LIMIT", "YAML expanded node budget exceeded.", n)
	}
	if s.active[n] {
		return nil, yamlError("E_YAML_ALIAS_CYCLE", "Cyclic YAML aliases are unsupported.", n)
	}
	s.active[n] = true
	defer delete(s.active, n)
	if n.Kind == yaml.AliasNode {
		if n.Alias == nil {
			return nil, yamlError("E_YAML_INVALID", "Undefined YAML alias.", n)
		}
		if aliases >= maxYAMLAliasDepth {
			return nil, yamlError("E_YAML_LIMIT", "YAML alias depth exceeds 127.", n)
		}
		return s.expand(n.Alias, depth, aliases+1)
	}
	if n.Kind == yaml.MappingNode || n.Kind == yaml.SequenceNode {
		if depth >= MaxJSONDepth {
			return nil, yamlError("E_YAML_LIMIT", "Expanded YAML container depth exceeds 127.", n)
		}
		depth++
	}
	v := &Value{}
	switch n.Kind {
	case yaml.MappingNode:
		if n.Tag != "!!map" {
			return nil, yamlError("E_YAML_UNSUPPORTED", "Unsupported YAML mapping tag.", n)
		}
		v.kind = Object
		seen := make(map[string]bool, len(n.Content)/2)
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Tag == "!!merge" {
				return nil, yamlError("E_YAML_UNSUPPORTED", "YAML merge keys are unsupported; write explicit members.", key)
			}
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || plainYAMLNumber(key) {
				return nil, yamlError("E_YAML_UNSUPPORTED", "YAML mapping keys must be string scalars.", key)
			}
			if seen[key.Value] {
				return nil, yamlError("E_YAML_INVALID", "Duplicate YAML mapping key.", key)
			}
			seen[key.Value] = true
			// Count key visits/bytes using the same scalar path as values.
			if _, err := s.expand(key, depth, aliases); err != nil {
				return nil, err
			}
			if err := s.addBytes(1, key); err != nil { // colon
				return nil, err
			}
			child, err := s.expand(n.Content[i+1], depth, aliases)
			if err != nil {
				return nil, err
			}
			v.members = append(v.members, Member{Name: key.Value, Value: child})
		}
		if len(v.members) > 8 {
			v.index = make(map[string]*Value, len(v.members))
			for _, member := range v.members {
				v.index[member.Name] = member.Value
			}
		} else {
			v.lookup = append([]Member(nil), v.members...)
		}
	case yaml.SequenceNode:
		if n.Tag != "!!seq" {
			return nil, yamlError("E_YAML_UNSUPPORTED", "Unsupported YAML sequence tag.", n)
		}
		v.kind = Array
		for _, item := range n.Content {
			child, err := s.expand(item, depth, aliases)
			if err != nil {
				return nil, err
			}
			v.items = append(v.items, child)
		}
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			// Preserve quoted/tagged strings; plain numeric spellings always
			// follow the finite JSON numeric profile, including resolver overflow.
			if plainYAMLNumber(n) {
				f, err := strconv.ParseFloat(n.Value, 64)
				if !jsonNumber.MatchString(n.Value) || err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
					return nil, yamlError("E_YAML_UNSUPPORTED", "YAML numbers must use finite JSON number syntax.", n)
				}
				v.kind, v.text = Number, n.Value
			} else {
				v.kind, v.text = String, n.Value
			}
		case "!!timestamp":
			if n.Style&yaml.TaggedStyle != 0 {
				return nil, yamlError("E_YAML_UNSUPPORTED", "Explicit timestamp tags are unsupported; use strings.", n)
			}
			// Implicit date/time-looking values remain strings, never time.Time.
			v.kind, v.text = String, n.Value
		case "!!bool":
			text := strings.ToLower(n.Value)
			if text != "true" && text != "false" {
				return nil, yamlError("E_YAML_UNSUPPORTED", "YAML booleans must be true or false.", n)
			}
			v.kind, v.boolean = Boolean, text == "true"
		case "!!null":
			text := strings.ToLower(n.Value)
			if text != "null" && text != "~" && text != "" {
				return nil, yamlError("E_YAML_UNSUPPORTED", "Unsupported YAML null spelling.", n)
			}
			v.kind = Null
		case "!!int", "!!float":
			if n.Tag == "!!int" && !jsonInteger.MatchString(n.Value) {
				return nil, yamlError("E_YAML_UNSUPPORTED", "YAML integer tags require JSON integer syntax.", n)
			}
			f, err := strconv.ParseFloat(n.Value, 64)
			if !jsonNumber.MatchString(n.Value) || err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
				return nil, yamlError("E_YAML_UNSUPPORTED", "YAML numbers must use finite JSON number syntax.", n)
			}
			v.kind, v.text = Number, n.Value
		default:
			return nil, yamlError("E_YAML_UNSUPPORTED", "Unsupported YAML scalar tag.", n)
		}
	default:
		return nil, yamlError("E_YAML_INVALID", "Unsupported YAML node.", n)
	}
	var size int
	switch v.kind {
	case Object:
		size = 2 + max(0, len(v.members)-1)
	case Array:
		size = 2 + max(0, len(v.items)-1)
	case String:
		encoded, _ := json.Marshal(v.text)
		size = len(encoded)
	case Number:
		size = len(v.text)
	case Boolean:
		size = 5
		if v.boolean {
			size = 4
		}
	case Null:
		size = 4
	}
	if err := s.addBytes(size, n); err != nil {
		return nil, err
	}
	return v, nil
}
