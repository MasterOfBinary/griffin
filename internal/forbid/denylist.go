package forbid

// Patterns are the forbidden paths this lab must not grow.
// This file is the only source file Scan skips: the denylist has to name the paths.
func Patterns() []string {
	return []string{
		"jup.ag",
		"jupiter",
		"sendtransaction",
		"signtransaction",
		"privatekey",
		"private_key",
		"private key",
		"secretkey",
		"secret_key",
		"secret key",
		"keypair",
		"wallet",
		"purchase",
		"begin private",
		"api_key",
		"api-key",
		"apikey",
	}
}

// BannedNames are repository paths that would pull in a container or a frontend build.
func BannedNames() []string {
	return []string{
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.yaml",
		"package.json",
		"package-lock.json",
	}
}
