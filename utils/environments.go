// Package utils contains common utility functions that can be shared across different projects.
package utils

import "os"

// GetEnvOrDefault retrieves the value of an environment variable identified by the key.
// If the variable is not present, it returns the provided default value.
// I find this helper useful for reducing boilerplate when handling application configuration.
func GetEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
