package cli

import "errors"

// Validate the entire declared inventory even for a partial update. Do not
// inspect unselected files/roots here. Legacy implicit layouts remain unchanged;
// opting into a shared package also considers peers' effective default package.
func validateRuntimeRoots(settings map[string]arguments) error {
	type binding struct {
		root               string
		explicit, conflict bool
	}
	packages := map[string]binding{}
	for _, a := range settings {
		pkg := a.runtimePackage
		if pkg == "" {
			pkg = a.packageName + ".models"
		}
		b, ok := packages[pkg]
		if !ok {
			b.root = a.output
		}
		b.explicit = b.explicit || a.runtimePackage != ""
		b.conflict = b.conflict || ok && b.root != a.output
		packages[pkg] = b
	}
	for _, b := range packages {
		if b.explicit && b.conflict {
			return errors.New("shared runtime roots differ")
		}
	}
	return nil
}
