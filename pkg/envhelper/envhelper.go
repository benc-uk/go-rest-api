package envhelper

import _ "github.com/joho/godotenv/autoload" // Autoload .env file
import "os"
import "strconv"

// Internal function to fetch environmental variable or return default
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

// GetEnvString is a simple helper function to read an environment or return a default value.
func GetEnvString(key string, defaultVal string) string {
	return getEnv(key, defaultVal)
}

// GetEnvInt is a simple helper function to read an environment or return a default value.
func GetEnvInt(key string, defaultVal int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

// GetEnvBool is a simple helper function to read an environment or return a default value.
func GetEnvBool(key string, defaultVal bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultVal
}