// Package document loads bounded JSON/YAML into an ordered, exact-number tree.
// It has no filesystem, environment, subprocess, or network dependencies.
package document

import (
	"bytes"
	"encoding/json"
)

// Kind distinguishes missing values from explicit JSON null.
type Kind uint8

const (
	Invalid Kind = iota
	Null
	Boolean
	Number
	String
	Array
	Object
)

// Member retains an object's original declaration order.
type Member struct {
	Name  string
	Value *Value
}

// Value owns an immutable-by-convention JSON tree. Numbers retain their original
// tokens; object lookup never determines iteration order. Only the loader builds
// values, and analyzers borrow them rather than expanding references by copying.
type Value struct {
	kind    Kind
	text    string
	boolean bool
	items   []*Value
	members []Member
	// Small objects retain an independent ordered lookup snapshot, just as
	// the large-object index is independent of the Members view.
	lookup []Member
	index  map[string]*Value
}

func (v *Value) Kind() Kind {
	if v == nil {
		return Invalid
	}
	return v.kind
}

func (v *Value) Get(name string) (*Value, bool) {
	if v.Kind() != Object {
		return nil, false
	}
	// Most schema nodes have only a handful of keys. Linear lookup avoids
	// allocating a hash table for each one; wide objects retain a hash index.
	if v.index == nil {
		for _, member := range v.lookup {
			if member.Name == name {
				return member.Value, true
			}
		}
		return nil, false
	}
	value, ok := v.index[name]
	return value, ok
}

func (v *Value) Text() (string, bool) {
	if v.Kind() != String {
		return "", false
	}
	return v.text, true
}

func (v *Value) Bool() (bool, bool) {
	if v.Kind() != Boolean {
		return false, false
	}
	return v.boolean, true
}

func (v *Value) Numeric() (json.Number, bool) {
	if v.Kind() != Number {
		return "", false
	}
	return json.Number(v.text), true
}

// Members and Items are read-only views; callers must not mutate their contents.
func (v *Value) Members() []Member { return v.members }
func (v *Value) Items() []*Value   { return v.items }

// MarshalJSON preserves declaration order and numeric tokens. It is useful for
// literal validation and tests, not a second normalization/expansion mechanism.
func (v *Value) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer
	if err := v.appendJSON(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (v *Value) appendJSON(out *bytes.Buffer) error {
	switch v.Kind() {
	case Invalid, Null:
		out.WriteString("null")
	case Boolean:
		if v.boolean {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case Number:
		out.WriteString(v.text)
	case String:
		encoded, err := json.Marshal(v.text)
		if err != nil {
			return err
		}
		out.Write(encoded)
	case Array:
		out.WriteByte('[')
		for i, item := range v.items {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := item.appendJSON(out); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case Object:
		out.WriteByte('{')
		for i, member := range v.members {
			if i > 0 {
				out.WriteByte(',')
			}
			key, err := json.Marshal(member.Name)
			if err != nil {
				return err
			}
			out.Write(key)
			out.WriteByte(':')
			if err := member.Value.appendJSON(out); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	}
	return nil
}
