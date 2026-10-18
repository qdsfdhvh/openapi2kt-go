package core_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const inlineHierarchyDocument = `{"Shape":{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"circle"},"r":{"type":"integer"}},"required":["kind","r"]},{"type":"object","properties":{"kind":{"type":"string","const":"square"},"side":{"type":"integer"}},"required":["kind","side"]}],"discriminator":{"propertyName":"kind"}}}`

// An inline branch carries no component name, so its discriminator constant
// supplies both the class name and the dispatch value. The emitted shape must
// match the `$ref` spelling exactly, so call sites are interchangeable.
func TestHierarchyInlineMembersEmitTheSameShapeAsReferences(t *testing.T) {
	inline := schemas(inlineHierarchyDocument)
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(inline, "Dto")
			r.Options.Validation, r.Options.ModelStyle = validation, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(validation, style, err)
			}
			for _, marker := range []struct{ file, want string }{
				{"ShapeDto", "public sealed class ShapeDto {"},
				{"CircleDto", `fromSerialName("circle")`},
				{"SquareDto", `fromSerialName("square")`},
				{"ShapeDto", `tag == "circle" -> org.example.models.models.CircleDto.serializer()`},
				{"ShapeDto", `tag == "square" -> org.example.models.models.SquareDto.serializer()`},
			} {
				if text := source(t, got, marker.file); !strings.Contains(text, marker.want) {
					t.Fatalf("%s lost %q:\n%s", marker.file, marker.want, text)
				}
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Inline hierarchy generation drifted", err)
			}
		}
	}
	// The class name comes from the constant, not the branch index: reordering
	// the branches must not rename a leaf.
	reordered := schemas(`{"Shape":{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"square"},"side":{"type":"integer"}},"required":["kind","side"]},{"type":"object","properties":{"kind":{"type":"string","const":"circle"},"r":{"type":"integer"}},"required":["kind","r"]}],"discriminator":{"propertyName":"kind"}}}`)
	a, err := core.GenerateWithDiagnostics(request(inline, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := core.GenerateWithDiagnostics(request(reordered, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"CircleDto", "SquareDto"} {
		if source(t, a, file) != source(t, b, file) {
			t.Fatalf("%s changed with branch order:\n%s\n---\n%s", file, source(t, a, file), source(t, b, file))
		}
	}
}

// A single-member String enum is the other admitted name source.
func TestHierarchyInlineMembersAcceptSingleMemberEnums(t *testing.T) {
	doc := schemas(`{"Shape":{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","enum":["circle"]},"r":{"type":"integer"}},"required":["kind","r"]}],"discriminator":{"propertyName":"kind"}}}`)
	got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "CircleDto")
	// A vocabulary leaf keeps the required typed constructor argument, exactly as
	// the `$ref` spelling does; only the constant form inlines a fixed value.
	if !strings.Contains(text, "public override val kind: org.example.models.models.ShapeDtoKind,") {
		t.Fatalf("Single-member enum lost its required tag argument:\n%s", text)
	}
	if !strings.Contains(source(t, got, "ShapeDto"), `tag == "circle" -> org.example.models.models.CircleDto.serializer()`) {
		t.Fatalf("Single-member enum did not reach dispatch:\n%s", source(t, got, "ShapeDto"))
	}
}

// Anything without a single constant keeps its existing rejection, at the same
// Pointer it reported before this contract.
func TestHierarchyInlineMembersKeepNaminglessBranchesRejected(t *testing.T) {
	for _, tc := range []struct{ name, doc, pointer string }{
		{"empty object", `{"Shape":{"oneOf":[{"type":"object"}],"discriminator":{"propertyName":"kind"}}}`, "/components/schemas/Shape/oneOf/0/type"},
		{"no discriminator property", `{"Shape":{"oneOf":[{"type":"object","properties":{"r":{"type":"integer"}}}],"discriminator":{"propertyName":"kind"}}}`, "/components/schemas/Shape/oneOf/0/type"},
		{"multi-member enum", `{"Shape":{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","enum":["a","b"]}},"required":["kind"]}],"discriminator":{"propertyName":"kind"}}}`, "/components/schemas/Shape/oneOf/0/type"},
		{"non-string tag", `{"Shape":{"oneOf":[{"type":"object","properties":{"kind":{"type":"integer","const":1}},"required":["kind"]}],"discriminator":{"propertyName":"kind"}}}`, "/components/schemas/Shape/oneOf/0/type"},
		{"primitive branch", `{"Shape":{"oneOf":[{"type":"string"}],"discriminator":{"propertyName":"kind"}}}`, "/components/schemas/Shape/oneOf/0/type"},
		{"nullable branch", `{"Shape":{"oneOf":[{"type":"object","nullable":true,"properties":{"kind":{"type":"string","const":"a"}},"required":["kind"]}],"discriminator":{"propertyName":"kind"}}}`, "/components/schemas/Shape/oneOf/0/type"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(tc.doc), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			var ds core.Diagnostics
			if !errors.As(err, &ds) || len(ds) == 0 {
				t.Fatalf("%s: expected a diagnostic, got %v", tc.name, err)
			}
			if ds[0].Code != "E_SCHEMA_UNSUPPORTED" {
				t.Fatalf("%s: expected E_SCHEMA_UNSUPPORTED, got %s", tc.name, ds[0].Code)
			}
			if ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer {
				t.Fatalf("%s: expected %s, got %v", tc.name, tc.pointer, ds[0].Pointer)
			}
			if len(got.Artifacts.Files) != 0 {
				t.Fatalf("%s: rejected input emitted files", tc.name)
			}
		}
	}
}

// Two inline branches that resolve to the same class name must fail rather than
// silently overwrite one another.
func TestHierarchyInlineMembersRejectDuplicateClassNames(t *testing.T) {
	doc := schemas(`{"Shape":{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"same"},"a":{"type":"integer"}},"required":["kind","a"]},{"type":"object","properties":{"kind":{"type":"string","const":"same"},"b":{"type":"integer"}},"required":["kind","b"]}],"discriminator":{"propertyName":"kind"}}}`)
	got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
	if err == nil || len(got.Artifacts.Files) != 0 {
		t.Fatalf("Duplicate inline leaf names were admitted: %v", err)
	}
}
