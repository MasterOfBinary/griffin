package safety

import "regexp"

// Patterns match key material in JSON, env assignments, PEM blocks, and seed phrases.
// Names may contain underscores and prefixes. A colon or equals sign may sit after a closing quote.
var (
	pemRE = regexp.MustCompile(`(?is)-----begin [a-z0-9 ]{0,40}-----.*?-----end [a-z0-9 ]{0,40}-----`)
	// Up to 24 words so a phrase is removed, not only the first word.
	mnemonicRE = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])(?:seed[ _-]?phrase|mnemonic)["']?[ \t]*[:=][ \t]*(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|(?:\S+[ \t]+){0,23}\S+)`)
	// Sensitive names include prefixes such as AWS_ and suffixes such as _KEY.
	// Quoted JSON values keep internal spaces; bare values stop at whitespace or a delimiter.
	assignRE = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])(?:[A-Za-z0-9]*[_-]?)?(?:api[ _-]?key|private[ _-]?key|secret[ _-]?key|client[ _-]?secret|access[ _-]?key|secret)[A-Za-z0-9_-]*["']?[ \t]*[:=][ \t]*(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\s,}\]]+)`)
)

// ContainsKeyMaterial reports whether s carries key-like material.
func ContainsKeyMaterial(s string) bool {
	return pemRE.MatchString(s) || mnemonicRE.MatchString(s) || assignRE.MatchString(s)
}

// Redact replaces key-like material, including PEM bodies and the rest of a seed phrase, with a fixed marker.
func Redact(s string) string {
	s = pemRE.ReplaceAllString(s, "[redacted]")
	s = mnemonicRE.ReplaceAllString(s, "[redacted]")
	s = assignRE.ReplaceAllString(s, "[redacted]")
	return s
}
