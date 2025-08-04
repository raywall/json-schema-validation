package utils

import "os"

// GetEnvOrDefault retrieves an environment variable or uses a default value
func GetEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
