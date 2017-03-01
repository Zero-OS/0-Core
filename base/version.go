package core

import "fmt"

/*
The constants in this file are auto-replaced with the actual values
during the build of both core0 and coreX (only using the make file)
*/

const (
	vBranch   = "{branch}"
	vRevision = "{revision}"
	vDirty    = "{dirty}"
)

type version struct{}

func (v *version) String() string {
	s := fmt.Sprintf("%s@%s", vBranch, vRevision)
	if vDirty != "" {
		s += "#dirty-repo"
	}

	return s
}

func Version() fmt.Stringer {
	return &version{}
}
