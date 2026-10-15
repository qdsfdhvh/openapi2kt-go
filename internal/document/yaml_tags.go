package document

import (
	"bytes"
	"net/url"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"

	"openapi-kt/internal/diagnostic"
)

// Node.Tag has lost the spelling's identity: both !!str and the custom
// verbatim URI !<!!str> become "!!str". Use the parser's node marks to inspect
// only actual node properties, never strings/comments or YAML structure.
// Source traversal budgets have already passed. One rune scan locates every
// mark, avoiding repeated scans of long flow lines for each node.
func checkYAMLTagIdentity(data []byte, root *yaml.Node) *diagnostic.Diagnostic {
	type position struct{ line, column int }
	marks := make(map[position]*yaml.Node)
	var collect func(*yaml.Node)
	collect = func(n *yaml.Node) {
		marks[position{n.Line, n.Column}] = n
		for _, child := range n.Content {
			collect(child)
		}
	}
	collect(root) // Content only; aliases point to previously recorded nodes.
	line, column, offset := 1, 1, 0
	// The upstream reader removes only the initial UTF-8 BOM before marking.
	if bytes.HasPrefix(data, []byte("\xef\xbb\xbf")) {
		offset = 3
	}
	for offset < len(data) {
		if n := marks[position{line, column}]; n != nil {
			if !standardYAMLTagProperty(data[offset:]) {
				return yamlError("E_YAML_UNSUPPORTED", "Unsupported YAML tag identity.", n)
			}
		}
		r, width := utf8.DecodeRune(data[offset:])
		offset += width
		if r == '\r' && offset < len(data) && data[offset] == '\n' {
			offset++
		}
		if yamlLineBreak(r) {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return nil
}

func yamlLineBreak(r rune) bool {
	return r == '\n' || r == '\r' || r == '\u0085' || r == '\u2028' || r == '\u2029'
}

func standardYAMLTagProperty(source []byte) bool {
	// Node properties may put an anchor before the tag. The upstream grammar
	// restricts anchor names to ASCII alphanumeric, underscore and hyphen.
	if len(source) > 0 && source[0] == '&' {
		i := 1
		for i < len(source) {
			c := source[i]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
				i++
			} else {
				break
			}
		}
		source = source[i:]
		// Separation can span whitespace, line breaks and comments.
		for len(source) > 0 {
			r, width := utf8.DecodeRune(source)
			if r == ' ' || r == '\t' || yamlLineBreak(r) {
				source = source[width:]
			} else if r == '#' {
				for len(source) > 0 {
					r, width = utf8.DecodeRune(source)
					if yamlLineBreak(r) {
						break
					}
					source = source[width:]
				}
			} else {
				break
			}
		}
	}
	if len(source) == 0 || source[0] != '!' {
		return true // No explicit tag property at this node.
	}
	if bytes.HasPrefix(source, []byte("!!")) {
		return true // Default standard handle; directives are preflight-rejected.
	}
	if bytes.HasPrefix(source, []byte("!<")) {
		end := bytes.IndexByte(source, '>')
		if end < 0 {
			return false
		}
		uri, err := url.PathUnescape(string(source[2:end]))
		return err == nil && strings.HasPrefix(uri, "tag:yaml.org,2002:")
	}
	// Preserve the bare non-specific ! tag. Local tags (including !%21str)
	// cannot gain standard identity through URI percent decoding.
	if len(source) == 1 {
		return true
	}
	r, _ := utf8.DecodeRune(source[1:])
	return r == ' ' || r == '\t' || yamlLineBreak(r)
}
