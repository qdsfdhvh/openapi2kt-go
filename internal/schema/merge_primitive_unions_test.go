package schema

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPrimitiveUnionMergeAliasGroupAndPassEligibility(t *testing.T) {
	a := &PrimitiveUnion{ClassName: "A", Kinds: []PrimitiveKind{PrimitiveString, PrimitiveInteger}, NullVariant: true}
	z := &PrimitiveUnion{ClassName: "Z", Kinds: []PrimitiveKind{PrimitiveString, PrimitiveInteger}, NullVariant: true}
	if dedupEligible(a) || !mergeEligible(a) || mergeEligible(&PrimitiveUnion{Object: &Model{ClassName: "Object"}}) {
		t.Fatal("merge expansion leaked into other options or object contexts")
	}
	p := &Plan{Models: []Declaration{a, z}}
	MergeIdenticalModels(p, "org.example.unionmerge", nil, func(string) bool { return false })
	alias, ok := p.Models[1].(*PrimitiveUnionAlias)
	if !ok || alias.ClassName != "Z" || alias.Canonical != "org.example.unionmerge.models.A" || !alias.NullVariant || !reflect.DeepEqual(alias.Kinds, a.Kinds) {
		t.Fatal("lost root/sibling alias inventory", p.Models[1])
	}
	before, _ := json.Marshal(p.Models)
	MergeIdenticalModels(p, "org.example.unionmerge", nil, func(string) bool { return false })
	after, _ := json.Marshal(p.Models)
	if string(before) != string(after) {
		t.Fatal("union alias group cannot replay merging")
	}
	z.Kinds[0] = PrimitiveBoolean
	if alias.Kinds[0] != PrimitiveString {
		t.Fatal("alias group shared losing declaration's mutable kind slice")
	}
}
