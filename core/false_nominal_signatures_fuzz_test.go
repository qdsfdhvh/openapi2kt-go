package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzFalseNominalSignatures(f *testing.F) {
	for _, seed := range [][4]uint16{{188, 316, 0, 0}, {189, 316, 0, 1}, {190, 316, 1, 0}, {191, 316, 1, 1}, {240, 18, 0, 0}, {241, 18, 1, 0}} {
		f.Add(seed[0], seed[1], seed[2], seed[3])
	}
	f.Fuzz(func(t *testing.T, width, packageBytes, style, required uint16) {
		count := int(width%243) + 1
		n := int(packageBytes%340) + 1
		rq := request(falseNominalSignatureInput(count, required%2 != 0), "Dto")
		rq.Options.ModelStyle = core.DataClass
		rq.Options.PackageName = "org.example." + strings.Repeat("x", n)
		if style%2 != 0 {
			rq.Options.ModelStyle = core.PlainClass
		}
		report, err := core.GenerateWithDiagnostics(rq)
		again, e := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(report, again) || !reflect.DeepEqual(err, e) {
			t.Fatal("nondeterministic descriptor limit", err, e)
		}
		parent := len(rq.Options.PackageName+".models.WideDto") + 2
		leaf := parent + len(".FalseValue")
		reject := count > 240 || 3+count*leaf > 65535 || rq.Options.ModelStyle == core.DataClass && 20+2*parent+count*leaf+(count+31)/32 > 65535
		if reject != (err != nil) {
			t.Fatalf("descriptor accept mismatch count=%d package=%d style=%s err=%v", count, n, rq.Options.ModelStyle, err)
		}
		if err != nil && (len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("partial output/wrong error", err)
		}
	})
}
