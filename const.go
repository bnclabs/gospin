// Specifies constants.

package failsafe

import (
	"path/filepath"
)

func snapshotFile(path string) string {
	return filepath.Join(path, "safedict.snapshot")
}
