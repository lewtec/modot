package configcue

import (
	"fmt"
	"regexp"
)

// cueIdentifier matches names that CUE can select with dot notation.
var cueIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidateModuleName rejects module names that CUE cannot select as fields.
// A module's options live at options.<name>, so a name like "base16-gtk"
// would force the verbose options["base16-gtk"] form. Reject it instead of
// silently rewriting the name: the config author picks a valid slug.
func ValidateModuleName(name string) error {
	if cueIdentifier.MatchString(name) {
		return nil
	}
	return fmt.Errorf("module name %q is not a CUE identifier: use letters, digits, and _, and do not start with a digit (for example base16_gtk instead of base16-gtk)", name)
}
