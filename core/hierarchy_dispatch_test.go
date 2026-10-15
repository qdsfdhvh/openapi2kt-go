package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func hierarchyDispatchInput(t *testing.T) []byte {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/hierarchy-dispatch.json")
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestHierarchySeveralImplicitFieldsAndReusedTagType(t *testing.T) {
	for _, suffix := range []string{"", "Dto"} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			req := request(hierarchyDispatchInput(t), suffix)
			req.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(req)
			if err != nil {
				t.Fatal(err)
			}
			multi := source(t, got, "Group"+suffix)
			for _, part := range []string{"public abstract val tag: org.example.models.models.Group" + suffix + "Tag", "public abstract val mode: org.example.models.models.Group" + suffix + "Mode", `obj["tag"]?.jsonPrimitive?.content == "one" && obj["mode"]?.jsonPrimitive?.content == "fast"`, `obj["tag"]?.jsonPrimitive?.content == "two" && obj["mode"]?.jsonPrimitive?.content == "slow"`, "UnknownGroup" + suffix} {
				if !strings.Contains(multi, part) {
					t.Fatal(part, multi)
				}
			}
			if strings.Contains(multi, "FirstDuplicate"+suffix+".serializer()") {
				t.Fatal("duplicate compound rule became an additional route", multi)
			}
			if strings.Index(multi, "First"+suffix+".serializer()") > strings.Index(multi, "Range"+suffix+".serializer()") || !strings.Contains(multi, `obj["tag"]?.jsonPrimitive?.content == "three"`) || !strings.Contains(multi, " || ") {
				t.Fatal("const-first/multienum compound dispatch lost", multi)
			}
			rangeLeaf := source(t, got, "Range"+suffix)
			for _, wire := range []string{"tag", "mode"} {
				if !strings.Contains(rangeLeaf, "public override val "+wire+": org.example.models.models.Group"+suffix+strings.ToUpper(wire[:1])+wire[1:]+",") {
					t.Fatal("enum must remain required typed constructor", rangeLeaf)
				}
			}
			missing := source(t, got, "ReusedMissingSecond"+suffix)
			if !strings.Contains(missing, `.Serializer.fromSerialName("two")`) || !strings.Contains(missing, "ReusedMissingRoot"+suffix+"Tag") {
				t.Fatal("nearest reused mapping lost", missing)
			}
			reused := source(t, got, "ReusedGroup"+suffix)
			if strings.Contains(reused, "public sealed class ReusedGroup"+suffix+"Tag") || strings.Count(reused, "public abstract override val tag: org.example.models.models.ReusedRoot"+suffix+"Tag") != 1 {
				t.Fatal(reused)
			}
			first := source(t, got, "ReusedFirst"+suffix)
			if !strings.Contains(first, "public override val tag: org.example.models.models.ReusedRoot"+suffix+"Tag") {
				t.Fatal(first)
			}
			again, err := core.GenerateWithDiagnostics(req)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic", err)
			}
		}
	}
}
