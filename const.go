// Constants.

package failsafe

import (
	"path/filepath"
)

// snapshotFile returns the file and its path to persist SafeDict on disk.
func snapshotFile(path string) string {
	return filepath.Join(path, "safedict.snapshot")
}
