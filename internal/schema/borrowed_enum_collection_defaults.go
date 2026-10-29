package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Shape discovery runs before the catalog exists. It retains only an original
// schema path; the actual borrowed backing and lexical enum must authorize the
// factory after reference/declaration planning.
func (a *analyzer) internalEnumCollectionDefaultSource(v *document.Value, pointer string) bool {
	if inlineStringEnumCollectionLeaf(v, 1) == nil {
		return false
	}
	p, original, ok := a.internalEnumFactoryLookup("#" + pointer)
	return ok && p == pointer && original == v
}

// An ordinary field owns its annotation even when a collection alias borrows
// its type. A child container never acquires an annotation role from this test.
func (a *analyzer) independentBorrowedEnumCollectionFieldDefault(v *document.Value, pointer string) bool {
	if !has(v, "default") || !a.internalEnumCollectionDefaultSource(v, pointer) {
		return false
	}
	i := strings.LastIndex(pointer, "/properties/")
	if i < 0 || strings.Contains(pointer[i+len("/properties/"):], "/") {
		return false
	}
	p := pointer[:i]
	_, owner, ok := a.componentSchemaAtReference("#" + p)
	if !ok {
		_, owner, ok = a.internalEnumFactoryLookup("#" + p)
	}
	return ok && directInternalModel(owner, p)
}

func (a *analyzer) admitInternalEnumCollectionFactory(c *component, source *wrapperDefaultSource) *diagnostic.Diagnostic {
	if !a.internalEnumCollectionDefaultSource(source.Schema, source.Pointer) {
		return schemaReferenceUnsupported(c.Pointer)
	}
	target, err := a.resolveSchema(text(get(c.Schema, "$ref")), c.Pointer)
	if err != nil {
		return err
	}
	if target.Schema != source.Schema || target.Pointer != source.Pointer {
		return schemaReferenceUnsupported(c.Pointer)
	}
	if err := a.bindBorrowedRootCollection(target, c.Pointer); err != nil {
		return err
	}
	if target.BorrowedCollection == nil || target.BorrowedCollection != a.borrowedCollections[source.Schema] {
		return schemaReferenceUnsupported(c.Pointer)
	}
	return a.captureTupleNativeSource(c.Schema, c.Pointer, source.Schema, source.Pointer)
}

func (a *analyzer) internalEnumCollectionFactoryNode(c *component, source *wrapperDefaultSource, validation *analyzer) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if err := a.admitInternalEnumFactory(c, source); err != nil {
		return nil, err
	}
	// The actual linked alias supplies null intent and all original checks. The
	// source walk supplies strict vocabulary and independent annotation roles.
	return validation.referencedDefaultSchemaRole(source.Schema, source.Pointer, a.componentType(c), 0, false, referencedCollectionOwnDefault)
}
