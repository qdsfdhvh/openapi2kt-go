package kotlin

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

type kotlinNameToken struct {
	start, end int
	text       string
	quoted     bool
}

// Generated Kotlin keeps its escaped helper/factory call names. DEX before
// version040 cannot encode whitespace in JVM method names, so those declarations
// receive deterministic ASCII JVM names. The lexical scan excludes comments,
// quoted values and raw strings: user defaults/docs must remain exact bytes.
// This is a JVM name annotation pass, never a schema/type/source rewrite.
func jvmFunctionNames(source string) string {
	tokens := kotlinNameTokens(source)
	used := map[string]bool{}
	for _, token := range tokens {
		used[token.text] = true
	}
	type insertion struct {
		at    int
		value string
	}
	insertions := []insertion{}
	assigned := map[string]string{}
	for i, token := range tokens {
		if token.text != "fun" || token.quoted {
			continue
		}
		for _, next := range tokens[i+1:] {
			if next.text == "(" {
				break
			}
			if !next.quoted {
				continue
			}
			if !strings.ContainsAny(next.text, " \t\r\n") {
				break
			}
			start := strings.LastIndexByte(source[:token.start], '\n') + 1
			indentEnd := start
			for indentEnd < token.start && (source[indentEnd] == ' ' || source[indentEnd] == '\t') {
				indentEnd++
			}
			// Emitters put every declaration on its own line. Do not alter a
			// future unexpected inline declaration or an existing JVM annotation.
			prefix := strings.TrimSpace(source[indentEnd:token.start])
			if prefix != "" && prefix != "public" && prefix != "private" && prefix != "internal" && prefix != "public inline" && prefix != "private inline" {
				break
			}
			previous := strings.TrimSpace(source[:start])
			if at := strings.LastIndexByte(previous, '\n'); strings.HasPrefix(strings.TrimSpace(previous[at+1:]), "@kotlin.jvm.JvmName(") {
				break
			}
			name := assigned[next.text]
			if name == "" {
				sum := sha256.Sum256([]byte(next.text))
				base := "openapiKtJvm_" + hex.EncodeToString(sum[:])
				name = base
				for suffix := 2; used[name]; suffix++ {
					name = base + "_" + strconv.Itoa(suffix)
				}
				used[name] = true
				assigned[next.text] = name
			}
			insertions = append(insertions, insertion{start, source[start:indentEnd] + "@kotlin.jvm.JvmName(\"" + name + "\")\n"})
			break
		}
	}
	if len(insertions) == 0 {
		return source
	}
	var out strings.Builder
	last := 0
	for _, item := range insertions {
		out.WriteString(source[last:item.at])
		out.WriteString(item.value)
		last = item.at
	}
	out.WriteString(source[last:])
	return out.String()
}

func kotlinNameTokens(source string) []kotlinNameToken {
	tokens := []kotlinNameToken{}
	for i := 0; i < len(source); {
		start := i
		switch {
		case strings.HasPrefix(source[i:], "//"):
			if end := strings.IndexByte(source[i:], '\n'); end >= 0 {
				i += end + 1
			} else {
				i = len(source)
			}
		case strings.HasPrefix(source[i:], "/*"):
			i += 2
			depth := 1
			for i < len(source) && depth > 0 {
				if strings.HasPrefix(source[i:], "/*") {
					depth++
					i += 2
				} else if strings.HasPrefix(source[i:], "*/") {
					depth--
					i += 2
				} else {
					i++
				}
			}
		case strings.HasPrefix(source[i:], `"""`):
			i += 3
			if end := strings.Index(source[i:], `"""`); end >= 0 {
				i += end + 3
			} else {
				i = len(source)
			}
		case source[i] == '"' || source[i] == '\'':
			quote := source[i]
			i++
			for i < len(source) {
				if source[i] == '\\' {
					i = min(i+2, len(source))
				} else if source[i] == quote {
					i++
					break
				} else {
					i++
				}
			}
		case source[i] == '`':
			i++
			if end := strings.IndexByte(source[i:], '`'); end >= 0 {
				i += end
				tokens = append(tokens, kotlinNameToken{start, i + 1, source[start+1 : i], true})
				i++
			} else {
				i = len(source)
			}
		case kotlinNameByte(source[i]):
			i++
			for i < len(source) && kotlinNameByte(source[i]) {
				i++
			}
			tokens = append(tokens, kotlinNameToken{start, i, source[start:i], false})
		default:
			i++
			if source[start] != ' ' && source[start] != '\t' && source[start] != '\n' && source[start] != '\r' {
				tokens = append(tokens, kotlinNameToken{start, i, source[start:i], false})
			}
		}
	}
	return tokens
}

func kotlinNameByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}
