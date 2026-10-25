package core_test

import (
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func FuzzActualFalseGenericSignatures(f *testing.F) {
	for _, seed := range [][4]uint16{{179, 316, 0, 0}, {180, 316, 0, 1}, {180, 316, 1, 0}, {181, 316, 1, 1}, {240, 18, 0, 0}, {241, 18, 1, 0}} {
		f.Add(seed[0], seed[1], seed[2], seed[3])
	}
	f.Fuzz(func(t *testing.T, width, pkgSize, style, required uint16) {
		count := int(width%243) + 1
		n := int(pkgSize%340) + 1
		input := falseGenericSignatureInput(count, required%2 != 0)
		before := append([]byte(nil), input...)
		rq := request(input, "Dto")
		rq.Options.PackageName = "org.example." + strings.Repeat("x", n)
		rq.Options.ModelStyle = core.DataClass
		if style%2 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		report, e := core.GenerateWithDiagnostics(rq)
		again, e2 := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(input, before) || !reflect.DeepEqual(report, again) || !reflect.DeepEqual(e, e2) {
			t.Fatal("source mutation/nondeterminism", e, e2)
		}
		self := len(rq.Options.PackageName+".models.WideDto") + 2
		leaf := self + len(".FalseValue")
		parameters := count * (len("Ljava/util/List<") + leaf + len(">;") + 1)
		reject := count > 240 || 3+parameters > 65535 || rq.Options.ModelStyle == core.DataClass && 2+self+parameters > 65535
		if reject != (e != nil) {
			t.Fatal("actualfalse generic boundary mismatch", count, n, style, e)
		}
		if e != nil && (len(report.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("partial/wrong generic refusal", e)
		}
	})
}
