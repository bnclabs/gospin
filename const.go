// Constants.

package failsafe

import (
    "path/filepath"
)

const HttpHdrNameLeader     = "go-failsafe-leader"
const HttpHdrNameLeaderAddr = "go-failsafe-leaderAddr"

// snapshotFile returns the file and its path to persist SafeDict on disk.
func snapshotFile(path string) string {
    return filepath.Join(path, "safedict.snapshot")
}
