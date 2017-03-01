package core

import "fmt"

/*
The constants in this file are auto-replaced with the actual values
during the build of both core0 and coreX (only using the make file)
*/

var (
	Branch   = "{branch}"
	Revision = "{revision}"
	Dirty    = "{dirty}"
)

type version struct{}

func (v *version) String() string {
	s := fmt.Sprintf("Version: %s @Revision: %s", Branch, Revision)
	if Dirty != "" {
		s += " (dirty-repo)"
	}

	return s
}

func Version() fmt.Stringer {
	return &version{}
}
