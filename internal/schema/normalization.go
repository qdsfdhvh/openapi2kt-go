package schema

import (
	"math"
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type nestedDecision struct {
	Scalar string
	Enum   []string
}

type normalizationBranch struct {
	Schema                *document.Value
	Pointer, Kind, Format string
	FormatPresent         bool
}

// normalizeNested inspects direct branches only. Root declarations and
// reference-owned pins have different callers and never enter these folds.
func (a *analyzer) normalizeNested(union *unionView, depth int) (*nestedDecision, *diagnostic.Diagnostic) {
	// Product alternatives already retain validated original object shapes.
	// Their shared root annotations are not scalar branch-folding inputs.
	if len(union.Branches) > 0 && union.Branches[0].Shape != nil {
		return nil, nil
	}
	if len(union.Branches) < 2 {
		return nil, nil
	}
	branches := make([]normalizationBranch, 0, len(union.Branches))
	for _, branch := range union.Branches {
		v, p := branch.Schema, branch.Pointer
		if has(v, "$ref") {
			return nil, nil
		}
		if err := depthCheck(depth, p); err != nil {
			return nil, err
		}
		kind, nullable, err := schemaType(v, p)
		if err != nil {
			return nil, err
		}
		if nullable && kind == "object" && !has(v, "allOf") && !has(v, "oneOf") && !has(v, "anyOf") {
			return nil, nil // Object dispatch preserves physical variants/null intent.
		}
		if nullable {
			key := "type"
			if b, ok := get(v, "nullable").Bool(); ok && b {
				key = "nullable"
			}
			return nil, unsupported(naming.Pointer(p, key))
		}
		format, present := "", has(v, "format")
		if present {
			var ok bool
			format, ok = get(v, "format").Text()
			if !ok {
				return nil, invalid("format must be a string.", naming.Pointer(p, "format"))
			}
		}
		branches = append(branches, normalizationBranch{Schema: v, Pointer: p, Kind: kind, Format: format, FormatPresent: present})
	}
	for _, branch := range branches {
		if branch.Format != branches[0].Format || branch.FormatPresent != branches[0].FormatPresent || has(branch.Schema, "const") {
			return nil, nil
		}
	}
	named, free, stringLike := false, false, true
	for _, branch := range branches {
		named = named || has(branch.Schema, "enum")
		free = free || !has(branch.Schema, "enum")
		stringLike = stringLike && (branch.Kind == "string" || branch.Kind == "" && has(branch.Schema, "enum"))
	}
	if named && free && stringLike {
		var vocabulary []string
		seen := map[string]bool{}
		for _, branch := range branches {
			if has(branch.Schema, "enum") {
				if _, err := buildStringEnum("Validation", branch.Schema, branch.Pointer); err != nil {
					return nil, err
				}
				for _, entry := range get(branch.Schema, "enum").Items() {
					wire := text(entry)
					if !seen[wire] {
						seen[wire] = true
						vocabulary = append(vocabulary, wire)
						if len(vocabulary) > MaxMembers {
							return nil, diagnostic.At("E_ENUM_LIMIT", "Merged enum exceeds the supported 240-member limit.", naming.Pointer(branch.Pointer, "enum"))
						}
					}
				}
			} else {
				if err := a.unionConstraints(branch); err != nil {
					return nil, err
				}
				if _, err := scalarMapping("string", branch.Schema, branch.Pointer); err != nil {
					return nil, err
				}
			}
		}
		a.warn("W_PRIMITIVE_UNION_APPROXIMATION", "A known string vocabulary plus a free string folds to an open enum, not exact oneOf match-count validation.", union.Pointer)
		return &nestedDecision{Enum: vocabulary}, nil
	}
	if named || branches[0].Kind != "string" && branches[0].Kind != "integer" && branches[0].Kind != "number" && branches[0].Kind != "boolean" {
		return nil, nil
	}
	for _, branch := range branches {
		if branch.Kind != branches[0].Kind {
			return nil, nil
		}
	}
	scalar, err := scalarMapping(branches[0].Kind, branches[0].Schema, branches[0].Pointer)
	if err != nil {
		return nil, err
	}
	for _, branch := range branches {
		if err := a.unionConstraints(branch); err != nil {
			return nil, err
		}
		if _, err := scalarMapping(branch.Kind, branch.Schema, branch.Pointer); err != nil {
			return nil, err
		}
		if branch.Kind == "integer" && branch.Format == "" {
			for _, key := range []string{"minimum", "maximum"} {
				if n, ok := floatLiteral(get(branch.Schema, key)); ok && (n < math.MinInt32 || n > math.MaxInt32) {
					scalar = "kotlin.Long"
				}
			}
		}
	}
	a.warn("W_PRIMITIVE_UNION_APPROXIMATION", "A same-kind union folds to its native scalar, not exact oneOf match-count validation.", union.Pointer)
	return &nestedDecision{Scalar: scalar}, nil
}

func (a *analyzer) unionConstraints(branch normalizationBranch) *diagnostic.Diagnostic {
	keys := []string{}
	switch branch.Kind {
	case "string":
		keys = []string{"pattern", "minLength", "maxLength"}
	case "integer", "number":
		keys = []string{"minimum", "maximum"}
	case "boolean":
	default:
		return unsupported(branch.Pointer)
	}
	allowed := append([]string{"type", "nullable", "format"}, keys...)
	if err := checkKeys(branch.Schema, allowed, branch.Pointer); err != nil {
		return err
	}
	for _, key := range keys {
		v, exists := branch.Schema.Get(key)
		if !exists {
			continue
		}
		p := naming.Pointer(branch.Pointer, key)
		valid := false
		switch key {
		case "pattern":
			valid = v.Kind() == document.String
		case "minLength", "maxLength":
			if n, ok := v.Numeric(); ok {
				_, err := strconv.ParseUint(string(n), 10, 64)
				valid = err == nil
			}
		default:
			_, valid = floatLiteral(v)
		}
		if !valid {
			return invalid("Constraint slot has an invalid scalar value.", p)
		}
		a.warn("W_UNION_CONSTRAINT_APPROXIMATION", "A folded primitive union retains its mapped type without enforcing this branch constraint.", p)
	}
	for _, pair := range [][2]string{{"minLength", "maxLength"}, {"minimum", "maximum"}} {
		low, high := get(branch.Schema, pair[0]), get(branch.Schema, pair[1])
		if low == nil || high == nil {
			continue
		}
		inconsistent := false
		if pair[0] == "minLength" {
			l, _ := low.Numeric()
			h, _ := high.Numeric()
			lv, _ := strconv.ParseUint(string(l), 10, 64)
			hv, _ := strconv.ParseUint(string(h), 10, 64)
			inconsistent = lv > hv
		} else {
			l, _ := floatLiteral(low)
			h, _ := floatLiteral(high)
			inconsistent = l > h
		}
		if inconsistent {
			return invalid("Constraint bounds are inconsistent.", naming.Pointer(branch.Pointer, pair[1]))
		}
	}
	return nil
}
