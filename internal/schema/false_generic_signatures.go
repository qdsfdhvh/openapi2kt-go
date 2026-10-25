package schema

// Generic signatures retain List/Map arguments. Only actual emitted false
// classifiers establish nominal lengths; unknown aliases and value classes
// cannot be expanded from their spelling or a speculative backing TypeRef.
func minimumFalseGenericSignature(t TypeRef, boxed bool, known map[string]int) (int, bool) {
	switch t.Kind {
	case Nullable:
		if t.Element != nil {
			return minimumFalseGenericSignature(*t.Element, true, known)
		}
	case List, Map:
		if t.Element == nil {
			return 0, false
		}
		child, found := minimumFalseGenericSignature(*t.Element, true, known)
		prefix := "Ljava/util/List<"
		if t.Kind == Map {
			prefix = "Ljava/util/Map<Ljava/lang/String;"
		}
		return len(prefix) + child + len(">;"), found
	case Named:
		if t.Element == nil {
			return known[t.Name], known[t.Name] > 0
		}
	case Scalar:
		return minimumJVMSignatureBytes(t, boxed), false
	}
	return 0, false
}

// An unbroken inline List chain ending an actual non-null false class uses a
// wildcard for each nested List and that abstract/nonfinal terminal class.
// Map, nullable inner types and aliases retain conservative zero variance.
func knownFalseListVarianceBytes(t TypeRef, known map[string]int) int {
	t = t.WithoutRootNull()
	variance := 0
	for t.Kind == List && t.Element != nil {
		t = *t.Element
		if t.Kind == List {
			variance++
			continue
		}
		if t.Kind == Named && t.Element == nil && known[t.Name] > 0 {
			return variance + 1
		}
		return 0
	}
	return 0
}
