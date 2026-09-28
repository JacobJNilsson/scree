// Package contract holds the types that several pipeline stages share, and it imports no other internal package.
package contract

// SourceSet names the group a file belongs to.
type SourceSet string

// The source sets of spec 002, in the order a report lists them.
const (
	Production  SourceSet = "production"
	Test        SourceSet = "test"
	Generated   SourceSet = "generated"
	Vendored    SourceSet = "vendored"
	Testdata    SourceSet = "testdata"
	Excluded    SourceSet = "excluded"
	Unsupported SourceSet = "unsupported"
)
