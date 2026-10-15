package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestNamedDTOChainRetainsCatalogSourceAndCachedWork(t *testing.T) {
	root, err := document.Load([]byte(`{"components":{"schemas":{"Owner":{"properties":{"first":{"$ref":"#/components/schemas/Owner/properties/last","nullable":false},"last":{"$ref":"#/components/schemas/Node","nullable":true}}},"Node":{"type":"object","nullable":true,"properties":{"id":{"type":"string"}}}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	schemas := get(get(root, "components"), "schemas")
	owner, node := get(schemas, "Owner"), get(schemas, "Node")
	c := &component{Wire: "Node", Name: "NodeDto", Schema: node, Pointer: "/components/schemas/Node", NullableDTO: true}
	a := analyzer{options: Options{PackageName: "org.example.named"}, components: map[string]*component{"Node": c, "Owner": {Wire: "Owner", Name: "OwnerDto", Schema: owner, Pointer: "/components/schemas/Owner"}}, aliasPlanning: true}
	a.indexInternalModels(nil)
	target, err := a.resolveSchema("#/components/schemas/Owner/properties/first", "/use")
	if err != nil || target.Catalog != c || target.Schema != node || target.Pointer != c.Pointer || target.Steps != 3 || target.Nullable == nil || *target.Nullable || !target.NullableBinding || len(a.components) != 2 {
		t.Fatal("named identity/source/work", target, err)
	}
	ty, err := a.referenceType(target, nil, 45, "/use")
	if err != nil || ty.Kind != Named || ty.SchemaSteps != 3 || ty.Name != "org.example.named.models.NodeDto" {
		t.Fatal("false backing/work", ty, err)
	}
	yes := true
	ty, err = a.referenceType(target, &yes, 45, "/use")
	if err != nil || ty.Kind != Nullable || ty.SchemaSteps != 3 {
		t.Fatal("nullable cached work", ty, err)
	}
	if _, err = a.referenceType(target, nil, 46, "/deep"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("depth", err)
	}
	borrowed, err := a.lookupSchemaReference("#/components/schemas/Owner/properties/first", "/borrowed")
	if err != nil {
		t.Fatal(err)
	}
	if resolved, err := a.resolveInternalReferenceChainAt(borrowed, "/borrowed", 0, map[*document.Value]bool{}, false); err != nil || resolved.Catalog != c || resolved.Schema != node || resolved.Steps != target.Steps {
		t.Fatal("original ordinaryDTO borrowed leaf catalog/source/hops lost", resolved, err)
	}
	a.indexInternalModels(map[string]*Hierarchy{"Node": {}})
	if _, err = a.resolveSchema("#/components/schemas/Owner/properties/first", "/hierarchy"); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("hierarchy nominal terminal admitted", err)
	}
}
