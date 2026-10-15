package schema

import (
	"openapi-kt/internal/diagnostic"
	"strings"
)

func (a *analyzer) runtimePackage() string {
	if a.options.RuntimePackage != "" {
		return a.options.RuntimePackage
	}
	return a.options.PackageName + ".models"
}

// Package-relative runtime reservations do not reserve those names in other
// model packages. Check classifiers as well as paths (PatchSerializer lives in
// Patch.kt). Legacy same-model-package guards retain their original pointers.
func (a *analyzer) runtimeCollisions(plan *Plan) *diagnostic.Diagnostic {
	check := func(pkg, name string) *diagnostic.Diagnostic {
		collision := false
		if plan.PatchPackage == pkg {
			collision = strings.EqualFold(name, "Patch") || strings.EqualFold(name, "PatchSerializer")
		}
		if plan.BigintPackage == pkg {
			collision = collision || strings.EqualFold(name, bigintSerializerName)
		}
		if !collision {
			return nil
		}
		if pkg == a.options.PackageName+".models" {
			for _, c := range a.components {
				if strings.EqualFold(c.Name, name) {
					return diagnostic.At("E_NAME_COLLISION", "A declaration conflicts with a generated runtime utility.", c.Pointer)
				}
			}
		}
		return diagnostic.New("E_NAME_COLLISION", "A declaration conflicts with a generated runtime utility in the same package.")
	}
	for _, d := range plan.Models {
		if err := check(a.options.PackageName+".models", d.Name()); err != nil {
			return err
		}
	}
	if plan.Client != nil {
		for _, d := range plan.Client.Models {
			if err := check(a.options.PackageName+".api.models", d.Name()); err != nil {
				return err
			}
		}
		if err := check(a.options.PackageName, plan.Client.Name); err != nil {
			return err
		}
		for _, g := range plan.Client.Groups {
			if err := check(a.options.PackageName+".api", g.Name); err != nil {
				return err
			}
		}
	}
	return nil
}
