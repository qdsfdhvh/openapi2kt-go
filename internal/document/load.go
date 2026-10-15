package document

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"openapi-kt/internal/diagnostic"
)

// MaxDocumentBytes applies to the entire input, including metadata and whitespace.
const MaxDocumentBytes = 8 * 1024 * 1024

// MaxJSONDepth matches the original 128-level decoder budget: the 128th
// simultaneously open container is rejected. Schema depth is a separate limit.
const MaxJSONDepth = 127

type InputFormat string

const (
	Auto InputFormat = "auto"
	JSON InputFormat = "json"
	YAML InputFormat = "yaml"
)

func DetectFormat(format InputFormat, name string) (InputFormat, *diagnostic.Diagnostic) {
	if format == Auto || format == "" {
		base := filepath.Base(name)
		// A leading dot alone is not an extension (same as the Rust loader).
		dot := strings.LastIndexByte(base, '.')
		if dot <= 0 {
			return JSON, nil
		}
		switch strings.ToLower(base[dot+1:]) {
		case "json":
			return JSON, nil
		case "yaml", "yml":
			return YAML, nil
		default:
			return "", diagnostic.New("E_INPUT_FORMAT_UNSUPPORTED", "Select an explicit input format for this extension.")
		}
	}
	if format != JSON && format != YAML {
		return "", diagnostic.New("E_INPUT_FORMAT_UNSUPPORTED", "Select a supported input format.")
	}
	return format, nil
}

// Load selects one parser and never retries malformed JSON as YAML. Neither
// parser includes payloads or source names in errors; all data slots are checked.
func Load(data []byte, format InputFormat, name string) (*Value, *diagnostic.Diagnostic) {
	selected, err := DetectFormat(format, name)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxDocumentBytes {
		return nil, diagnostic.New("E_INPUT_LIMIT", "Document exceeds the 8 MiB size limit.")
	}
	if selected == YAML {
		return parseYAML(data)
	}
	return ParseObject(data, MaxDocumentBytes)
}

// ParseObject shares strict ordered JSON parsing with output metadata. Callers
// own their size policy; schema inputs retain the fixed Load limit above.
func ParseObject(data []byte, maxBytes int) (*Value, *diagnostic.Diagnostic) {
	if len(data) > maxBytes {
		return nil, diagnostic.New("E_INPUT_LIMIT", "JSON input exceeds the supported size limit.")
	}
	if !utf8.Valid(data) {
		return nil, invalidJSON(data, 0)
	}
	// encoding/json replaces unpaired escaped surrogates with U+FFFD. Reject
	// those before decoding so wire names and string literals cannot change.
	if offset, ok := validStringEscapes(data); !ok {
		return nil, invalidJSON(data, int64(offset))
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := readValue(decoder, 0)
	if err == nil {
		_, err = decoder.Token()
		if err == io.EOF {
			err = nil
		} else if err == nil {
			err = errInvalidJSON
		}
	}
	if err != nil {
		offset := decoder.InputOffset()
		if syntax, ok := err.(*json.SyntaxError); ok {
			offset = syntax.Offset - 1
		}
		return nil, invalidJSON(data, offset)
	}
	if value.Kind() != Object {
		return nil, diagnostic.At("E_DOCUMENT_INVALID", "The document root must be an object.", "")
	}
	return value, nil
}

// This private error is never exposed to callers; only the safe diagnostic is.
var errInvalidJSON = fmt.Errorf("invalid JSON tree")

func readValue(decoder *json.Decoder, depth int) (*Value, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch token := token.(type) {
	case nil:
		return &Value{kind: Null}, nil
	case bool:
		return &Value{kind: Boolean, boolean: token}, nil
	case string:
		return &Value{kind: String, text: token}, nil
	case json.Number:
		// Preserve the exact token, but retain the original rejection of
		// numbers outside finite JSON numeric range (e.g. 1e400).
		f, err := strconv.ParseFloat(string(token), 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, errInvalidJSON
		}
		return &Value{kind: Number, text: string(token)}, nil
	case json.Delim:
		if depth >= MaxJSONDepth {
			return nil, errInvalidJSON
		}
		switch token {
		case '{':
			v := &Value{kind: Object}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, errInvalidJSON
				}
				if _, duplicate := v.Get(key); duplicate {
					return nil, errInvalidJSON
				}
				child, err := readValue(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				// Keep small objects allocation-light without making duplicate
				// detection or lookup quadratic for wide component/property maps.
				if v.index == nil && len(v.members) == 8 {
					v.index = make(map[string]*Value, 16)
					for _, member := range v.members {
						v.index[member.Name] = member.Value
					}
				}
				if v.index != nil {
					v.index[key] = child
				}
				v.members = append(v.members, Member{Name: key, Value: child})
				if v.index == nil {
					v.lookup = v.members // Loader-only view for duplicate detection.
				} else {
					v.lookup = nil
				}
			}
			end, err := decoder.Token()
			if err != nil || end != json.Delim('}') {
				return nil, errInvalidJSON
			}
			if v.index == nil {
				v.lookup = append([]Member(nil), v.members...)
			}
			return v, nil
		case '[':
			v := &Value{kind: Array}
			for decoder.More() {
				child, err := readValue(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				v.items = append(v.items, child)
			}
			end, err := decoder.Token()
			if err != nil || end != json.Delim(']') {
				return nil, errInvalidJSON
			}
			return v, nil
		}
	}
	return nil, errInvalidJSON
}

func invalidJSON(data []byte, offset int64) *diagnostic.Diagnostic {
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	line, column := 1, 1
	for _, b := range data[:offset] {
		if b == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return diagnostic.New("E_JSON_INVALID", fmt.Sprintf("Invalid, duplicate-key, or overly nested JSON at line %d, column %d.", line, column))
}

func validStringEscapes(data []byte) (int, bool) {
	inString := false
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			inString = !inString
		} else if inString && data[i] == '\\' {
			if i+1 >= len(data) {
				return i, false
			}
			if data[i+1] != 'u' {
				i++ // including escaped quotes/backslashes; decoder checks legality
				continue
			}
			unit, ok := escapeUnit(data, i)
			if !ok {
				return i, false
			}
			if unit >= 0xdc00 && unit <= 0xdfff {
				return i, false
			}
			if unit >= 0xd800 && unit <= 0xdbff {
				low, ok := escapeUnit(data, i+6)
				if !ok || low < 0xdc00 || low > 0xdfff {
					return i, false
				}
				i += 6
			}
			i += 5
		}
	}
	return len(data), true // truncated strings are rejected by the decoder
}

func escapeUnit(data []byte, index int) (uint64, bool) {
	if index+6 > len(data) || data[index] != '\\' || data[index+1] != 'u' {
		return 0, false
	}
	unit, err := strconv.ParseUint(string(data[index+2:index+6]), 16, 16)
	return unit, err == nil
}
