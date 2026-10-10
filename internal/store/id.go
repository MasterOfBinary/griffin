package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
)

// idPattern keeps run ids usable as file names.
var idPattern = regexp.MustCompile(`^[a-z0-9._-]{8,160}$`)

// RunID derives the stored id from mode, configuration version, and dataset hash.
// The same three inputs always yield the same id. Any change yields a new id.
func RunID(mode, configVersion, datasetHash string) (string, error) {
	if mode == "" || configVersion == "" || datasetHash == "" {
		return "", fmt.Errorf("run id requires mode, configuration version, and dataset hash")
	}
	sum := sha256.Sum256([]byte(mode + "\n" + configVersion + "\n" + datasetHash))
	id := fmt.Sprintf("%s-v%s-%s", mode, configVersion, hex.EncodeToString(sum[:8]))
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("derived run id %q is not path-safe; use a simple configuration version", id)
	}
	return id, nil
}
