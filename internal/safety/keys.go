package safety

import "regexp"

// keyPatterns match material that must not land in fixtures, the store, logs, errors, or the UI.
var keyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)-----begin [a-z ]{0,40}-----`),
	regexp.MustCompile(`(?i)\b(api[_-]?key|secret|private[_-]?key|mnemonic|seed phrase)\b\s*[:=]\s*\S+`),
}

// ContainsKeyMaterial reports whether s carries key-like material.
func ContainsKeyMaterial(s string) bool {
	for _, re := range keyPatterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// Redact replaces key-like material with a fixed marker.
func Redact(s string) string {
	for _, re := range keyPatterns {
		s = re.ReplaceAllString(s, "[redacted]")
	}
	return s
}
