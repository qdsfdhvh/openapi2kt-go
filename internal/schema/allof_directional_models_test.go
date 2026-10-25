package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func TestAllOfDirectionalUsesValidatedWinningSourcesWithoutNewAuthority(t *testing.T) {
	data, readErr := os.ReadFile("../../tests/fixtures/http/allof-directional-models.json")
	if readErr != nil {
		t.Fatal(readErr)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.allofdirectional", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, modelDiagnosticPointers: map[*Model]string{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	client, e := a.clientPlan(root)
	if e != nil {
		t.Fatal(e)
	}
	if e := a.validateInternalModels(ds); e != nil {
		t.Fatal(e)
	}
	var original *Model
	for _, d := range ds {
		if m, ok := d.(*Model); ok && m.ClassName == "AccountDto" {
			original = m
		}
	}
	if original == nil {
		t.Fatal("missingoriginal")
	}
	source, ok := a.directionalModelSources[original]
	owner := a.components["Account"]
	if !ok || source.Schema != owner.Schema || source.Pointer != owner.Pointer || !source.Required["id"] || !source.Required["password"] || !source.Required["name"] || !source.Required["mode"] {
		t.Fatal("validatedmerged source/required lost", source)
	}
	fields := map[string]field{}
	for _, f := range source.Fields {
		fields[f.Wire] = f
	}
	first := get(owner.Schema, "allOf").Items()[0]
	second := get(owner.Schema, "allOf").Items()[1]
	if fields["id"].Schema != get(get(first, "properties"), "id") || fields["id"].Pointer != owner.Pointer+"/allOf/0/properties/id" || fields["password"].Schema != get(get(second, "properties"), "password") {
		t.Fatal("source fields reconstructed or pointer drift")
	}
	work, bytes := a.intersectionCaptures.Work, a.intersectionCaptures.Bytes
	plan := &Plan{Models: ds, Client: client, EmissionHints: a.emissionHints}
	if e := a.directionalModels(plan); e != nil {
		t.Fatal(e)
	}
	if a.intersectionCaptures.Work != work || a.intersectionCaptures.Bytes != bytes {
		t.Fatal("projection rewalked/admitted original source")
	}
	clones := 0
	for _, d := range plan.Client.Models {
		if m, ok := d.(*Model); ok && m.Directional {
			clones++
			if a.modelOrigins[m] != nil || a.directionalModelSources[m].Schema != nil {
				t.Fatal("directionalcopy acquired originalsource authority")
			}
		}
	}
	// operationDeclarations may retain operation-owned models under operations.
	for _, g := range plan.Client.Groups {
		for _, op := range g.Operations {
			for _, d := range op.Models {
				if m, ok := d.(*Model); ok && m.Directional {
					clones++
					if a.modelOrigins[m] != nil || a.directionalModelSources[m].Schema != nil {
						t.Fatal("operationcopy acquired sourceauthority")
					}
				}
			}
		}
	}
	if clones == 0 {
		t.Fatal("noactual directionalcopies")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("projection rewrote originalAST")
	}
	if len(original.Properties) != len(source.Fields) {
		t.Fatal("sharedcomponent fields were pruned")
	}
}
