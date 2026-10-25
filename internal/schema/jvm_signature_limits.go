package schema

import "openapi-kt/internal/diagnostic"

const maxJVMUTF8ConstantBytes = 65535

// This is a one-sided proof, not a JVM signature encoder. Omit wildcard
// variance and unknown classifier/alias spellings: both can only add bytes to
// the known inline container signature. Reject only an already impossible
// retained constructor, without erasing public types or lowering existing caps.
func minimumJVMSignatureBytes(ty TypeRef, boxed bool) int {
	switch ty.Kind {
	case Nullable:
		if ty.Element != nil {
			return minimumJVMSignatureBytes(*ty.Element, true)
		}
	case List, Map:
		if ty.Element == nil {
			return 0
		}
		prefix := "Ljava/util/List<"
		if ty.Kind == Map {
			prefix = "Ljava/util/Map<Ljava/lang/String;"
		}
		return len(prefix) + minimumJVMSignatureBytes(*ty.Element, true) + len(">;")
	case Scalar:
		if ty.Name == "kotlin.String" {
			return len("Ljava/lang/String;")
		}
		if ty.Name == "kotlin.ByteArray" {
			return len("[B")
		}
		primitive, wrapper := "", ""
		switch ty.Name {
		case "kotlin.Boolean":
			primitive, wrapper = "Z", "Ljava/lang/Boolean;"
		case "kotlin.Int":
			primitive, wrapper = "I", "Ljava/lang/Integer;"
		case "kotlin.Long":
			primitive, wrapper = "J", "Ljava/lang/Long;"
		case "kotlin.Float":
			primitive, wrapper = "F", "Ljava/lang/Float;"
		case "kotlin.Double":
			primitive, wrapper = "D", "Ljava/lang/Double;"
		}
		if boxed {
			return len(wrapper)
		}
		return len(primitive)
	}
	return 0
}

// Public Kotlin constructor parameters emit a wildcard for each nested List
// classifier in this exact inline chain. Keep this proof separate from the
// fixed syntax count: aliases, maps, nullable elements and unknown terminals
// contribute no variance bytes here. No uncertain classifier is expanded.
func knownListChainVarianceBytes(ty TypeRef) int {
	ty = ty.WithoutRootNull()
	variance := 0
	for ty.Kind == List && ty.Element != nil {
		ty = *ty.Element
		if ty.Kind == List {
			variance++
			continue
		}
		if ty.Kind == Scalar {
			switch ty.Name {
			case "kotlin.String", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double":
				return variance
			}
		}
		return 0
	}
	return 0
}

func (a *analyzer) certainJVMSignatureOverflow(plan *Plan) *diagnostic.Diagnostic {
	return firstRetainedModelFailure(plan, func(d *Model) *diagnostic.Diagnostic {
		bytes := len("()V")
		for _, p := range d.Properties {
			bytes += minimumJVMSignatureBytes(p.Type, false) + knownListChainVarianceBytes(p.Type)
			if bytes > maxJVMUTF8ConstantBytes {
				return diagnostic.At("E_MODEL_LIMIT", "The constructor's minimum generic signature exceeds the JVM UTF-8 limit; split the model or introduce named DTO boundaries.", a.modelPointers[d])
			}
		}
		return nil
	})
}

// CheckRetainedDataCopySignatureLimits runs after final model style and shape
// transformations. The legacy generic bound counts the validated ASCII
// package return prefix. Actual retained false classes refine generic and
// erased bounds separately. Plain classes do not own a Data copy API.
func CheckRetainedDataCopySignatureLimits(plan *Plan, packageName string) *diagnostic.Diagnostic {
	if err := firstRetainedModelFailure(plan, func(d *Model) *diagnostic.Diagnostic {
		if d.Plain || len(d.Properties) == 0 {
			return nil
		}
		bytes := len("()L;") + len(packageName)
		for _, p := range d.Properties {
			bytes += minimumJVMSignatureBytes(p.Type, false) + knownListChainVarianceBytes(p.Type)
		}
		if bytes > maxJVMUTF8ConstantBytes {
			return diagnostic.At("E_MODEL_LIMIT", "The retained Data copy method's minimum signature exceeds the JVM UTF-8 limit; split the model or shorten the package name.", plan.modelDiagnosticPointers[d])
		}
		return nil
	}); err != nil {
		return err
	}
	return checkFalseNominalSignatureLimits(plan, packageName)
}

func firstRetainedModelFailure(plan *Plan, check func(*Model) *diagnostic.Diagnostic) *diagnostic.Diagnostic {
	seen := map[Declaration]bool{}
	var visit func(Declaration) *diagnostic.Diagnostic
	visit = func(declaration Declaration) *diagnostic.Diagnostic {
		if seen[declaration] {
			return nil
		}
		seen[declaration] = true
		switch d := declaration.(type) {
		case *Model:
			if err := check(d); err != nil {
				return err
			}
			for _, nested := range d.Nested {
				if err := visit(nested); err != nil {
					return err
				}
			}
		case *ObjectUnion:
			for _, variant := range d.Variants {
				if err := visit(variant); err != nil {
					return err
				}
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				return visit(d.Object)
			}
		case *Hierarchy:
			if d.Leaf != nil {
				return visit(d.Leaf)
			}
		}
		return nil
	}
	for _, d := range plan.Models {
		if err := visit(d); err != nil {
			return err
		}
	}
	if plan.Client != nil {
		for _, d := range plan.Client.Models {
			if err := visit(d); err != nil {
				return err
			}
		}
	}
	return nil
}
