package schema

// Complete groups retain the original winner and every discarded occurrence.
// This is an authority check, not a new source expansion or a pointer-only grant.
func completePrimitiveContributions(s *shape, winner field, fields []field) bool {
	if len(fields) < 2 || fields[len(fields)-1] != winner {
		return false
	}
	remaining := map[field]int{}
	for _, f := range s.Discarded {
		if f.Wire == winner.Wire {
			remaining[f]++
		}
	}
	for _, f := range fields[:len(fields)-1] {
		if remaining[f] == 0 {
			return false
		}
		remaining[f]--
	}
	for _, count := range remaining {
		if count != 0 {
			return false
		}
	}
	return true
}
