package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestInternalDTOChainOriginalTerminalAndWork(t *testing.T) {
	root, err := document.Load([]byte(`{"components":{"schemas":{"Owner":{"properties":{"first":{"$ref":"#/components/schemas/Owner/properties/last","nullable":false},"last":{"$ref":"#/components/schemas/Owner/properties/node","nullable":true},"node":{"nullable":true,"properties":{"id":{"type":"string"}}}}}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	owner := get(get(get(root, "components"), "schemas"), "Owner")
	c := &component{Wire: "Owner", Name: "OwnerDto", Schema: owner, Pointer: "/components/schemas/Owner"}
	a := analyzer{options: Options{PackageName: "org.example.chains"}, components: map[string]*component{"Owner": c}, aliasPlanning: true}
	a.indexInternalModels(nil)
	target, err := a.resolveSchema("#/components/schemas/Owner/properties/first", "/use")
	if err != nil {
		t.Fatal(err)
	}
	if target.Schema != get(get(owner, "properties"), "node") || target.Catalog != nil || len(a.components) != 1 || target.Pointer != "/components/schemas/Owner/properties/node" || target.Steps != 3 || target.Nullable == nil || *target.Nullable {
		t.Fatal("terminal source/nearest false/source hops/catalog", target)
	}
	ty, err := a.referenceType(target, nil, 45, "/use")
	if err != nil || ty.Kind != Named || ty.SchemaSteps != 3 || ty.Name != "org.example.chains.models.OwnerDto.Node" {
		t.Fatal("nominal work lost", ty, err)
	}
	yes := true
	ty, err = a.referenceType(target, &yes, 45, "/use")
	if err != nil || ty.Kind != Nullable || ty.Element.SchemaSteps != 3 {
		t.Fatal("nullable erased cached work", ty, err)
	}
	if _, err = a.referenceType(target, nil, 46, "/deep"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("consumer reset chain height", err)
	}
	// This original ordinary DTO terminal now has a separate borrowed-leaf role.
	borrowed, err := a.lookupSchemaReference("#/components/schemas/Owner/properties/first", "/borrowed")
	if err != nil {
		t.Fatal(err)
	}
	if resolved, err := a.resolveInternalReferenceChainAt(borrowed, "/borrowed", 0, map[*document.Value]bool{}, false); err != nil || resolved.Schema != target.Schema || resolved.Steps != target.Steps {
		t.Fatal("original ordinaryDTO borrowed leaf source lost", resolved, err)
	}
}
