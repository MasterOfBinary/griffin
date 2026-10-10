package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
)

// idPattern keeps run ids usable as file names.
var idPattern = regexp.MustCompile(`^[a-z0-9._-]{8,160}$`)

// RunID derives the stored id from mode, configuration version, dataset hash, and a
// fingerprint of the research configuration. The same inputs always yield the same id.
func RunID(mode, configVersion, datasetHash, configFingerprint string) (string, error) {
	if mode == "" || configVersion == "" || datasetHash == "" || configFingerprint == "" {
		return "", fmt.Errorf("run id requires mode, configuration version, dataset hash, and config fingerprint")
	}
	sum := sha256.Sum256([]byte(mode + "\n" + configVersion + "\n" + datasetHash + "\n" + configFingerprint))
	id := fmt.Sprintf("%s-v%s-%s", mode, configVersion, hex.EncodeToString(sum[:8]))
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("derived run id %q is not path-safe; use a simple configuration version", id)
	}
	return id, nil
}
