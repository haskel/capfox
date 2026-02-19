package config

import (
	"os"
	"regexp"
)

var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

func substituteEnvVars(content []byte) []byte {
	return envVarRegex.ReplaceAllFunc(content, func(match []byte) []byte {
		varName := string(envVarRegex.FindSubmatch(match)[1])
		if value, exists := os.LookupEnv(varName); exists {
			return []byte(value)
		}
		return match
	})
}
