package schema

import "openapi-kt/internal/diagnostic"

// Only actual emitted false classes establish nominal descriptor costs. Other
// Named references may be aliases or value classes, so their spelling alone is
// insufficient. Generic and erased bounds remain separate; synthetic
// serializer/default constructors are not covered by the generic proof.
func checkFalseNominalSignatureLimits(plan *Plan, pkg string) *diagnostic.Diagnostic {
	falseTypes := map[string]int{}
	models := map[*Model]int{}
	var collect func(Declaration, string)
	collect = func(d Declaration, owner string) {
		name := owner + "." + d.Name()
		switch d := d.(type) {
		case *FalseValue:
			// Dots become either slash or dollar separators in JVM binary names.
			// Raw UTF-8 is also a lower bound for JVM modified UTF-8.
			falseTypes[name] = len(name) + 2
		case *Model:
			models[d] = len(name) + 2
			for _, nested := range d.Nested {
				collect(nested, name)
			}
		}
	}
	for _, d := range plan.Models {
		collect(d, pkg+".models")
	}
	if plan.Client != nil {
		for _, d := range plan.Client.Models {
			collect(d, pkg+".api.models")
		}
	}
	return firstRetainedModelFailure(plan, func(m *Model) *diagnostic.Diagnostic {
		self := models[m]
		if self == 0 {
			return nil
		}
		parameters, hasFalse := 0, false
		genericParameters, hasFalseGeneric := 0, false
		for _, p := range m.Properties {
			ty := p.Type.WithoutRootNull()
			if ty.Kind == Named && ty.Element == nil && falseTypes[ty.Name] > 0 {
				hasFalse = true
			}
			parameters += minimumFalseNominalDescriptor(p.Type, falseTypes)
			generic, known := minimumFalseGenericSignature(p.Type, false, falseTypes)
			genericParameters += generic + knownListChainVarianceBytes(p.Type) + knownFalseListVarianceBytes(p.Type, falseTypes)
			hasFalseGeneric = hasFalseGeneric || known
		}
		if hasFalseGeneric {
			if len("()V")+genericParameters > maxJVMUTF8ConstantBytes {
				return diagnostic.At("E_MODEL_LIMIT", "The constructor's known false-class generic signature exceeds the JVM UTF-8 limit; split the model or shorten names.", plan.modelDiagnosticPointers[m])
			}
			if !m.Plain && len("()")+self+genericParameters > maxJVMUTF8ConstantBytes {
				return diagnostic.At("E_MODEL_LIMIT", "The retained Data copy method's known false-class generic signature exceeds the JVM UTF-8 limit; split the model, shorten names or use Plain style.", plan.modelDiagnosticPointers[m])
			}
		}
		if !hasFalse {
			return nil
		}
		if len("()V")+parameters > maxJVMUTF8ConstantBytes {
			return diagnostic.At("E_MODEL_LIMIT", "The constructor's known false-class descriptor exceeds the JVM UTF-8 limit; split the model or shorten the package name.", plan.modelDiagnosticPointers[m])
		}
		// Data copy$default has an explicit receiver and return classifier, mask
		// ints for every property (including required fields), and an Object marker.
		if !m.Plain && len("()Ljava/lang/Object;")+2*self+parameters+(len(m.Properties)+31)/32 > maxJVMUTF8ConstantBytes {
			return diagnostic.At("E_MODEL_LIMIT", "The retained Data copy default method's known false-class descriptor exceeds the JVM UTF-8 limit; split the model, shorten the package name or use Plain style.", plan.modelDiagnosticPointers[m])
		}
		return nil
	})
}
func minimumFalseNominalDescriptor(t TypeRef, known map[string]int) int {
	switch t.Kind {
	case Nullable:
		if t.Element != nil {
			inner := *t.Element
			if inner.Kind == Scalar {
				return minimumJVMSignatureBytes(inner, true)
			}
			return minimumFalseNominalDescriptor(inner, known)
		}
	case Named:
		if t.Element == nil {
			return known[t.Name]
		}
	case List:
		return len("Ljava/util/List;")
	case Map:
		return len("Ljava/util/Map;")
	case Scalar:
		return minimumJVMSignatureBytes(t, false)
	}
	return 0
}
