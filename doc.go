/*
Package valid provides JSON Schema validation functionality for HTTP requests and JSON data.

This library offers a simple and flexible interface for validating JSON data against JSON Schema Draft 7 schemas,
with specialized support for web applications and APIs.

# Main Features

- Validation of JSON data against JSON Schema Draft 7
- HTTP middleware for automatic request validation
- Support for multiple validators for different endpoints
- Detailed and structured error messages
- Easy integration with existing Go applications
- Validation of files, strings, bytes, and interfaces

# Basic Usage

Create a validator from a schema file:

validator, err := validator.New("user-schema.json")

	if err != nil {
		log.Fatal(err)
	}

Create a validator from a JSON string:

	schemaJSON := `{
		"type": "object",
		"properties": {
			"name": {"type": "string", "minLength": 2},
			"email": {"type": "string", "format": "email"}
		},
		"required": ["name", "email"]
	}`

validator, err := validator.NewFromString(schemaJSON)

	if err != nil {
		log.Fatal(err)
	}

# Data Validation

Validate a JSON string:

result, err := validator.ValidateString(`{"name": "João", "email": "joao@exemplo.com"}`)

	if err != nil {
		log.Fatal(err)
	}

	if !result.Valid {
		for _, err := range result.Errors {
			fmt.Printf("Field: %s, Error: %s\n", err.Field, err.Message)
		}
	}

Validate an HTTP request:

	func userHandler(w http.ResponseWriter, r *http.Request) {
		result, err := validator.ValidateRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !result.Valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Invalid data",
				"details": result.Errors,
			})
			return
		}

		// Process valid data...
	}

#Middleware HTTP

Use as middleware for automatic validation:

http.HandleFunc("/users", validator.Middleware(userHandler))

Middleware with custom settings:

	config := validator.MiddlewareConfig{
		SkipMethods: []string{"GET", "DELETE"},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, result *validator.ValidationResult) {
			// Custom error handler
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors": result.Errors,
			})
		},
	}

http.HandleFunc("/users", validator.MiddlewareWithConfig(config, userHandler))

# Multiple Validators

For applications with multiple endpoints and different schemas:

multiValidator := validator.NewMultiValidator()

// Add validators
err := multiValidator.AddFromFile("user", "schemas/user.json")

	if err != nil {
		log.Fatal(err)
	}

err = multiValidator.AddFromString("product", productSchema)

	if err != nil {
		log.Fatal(err)
	}

// Use specific validators
userValidator, exists := multiValidator.Get("user")

	if exists {
		result, err := userValidator.ValidateString(jsonData)
		// ...
	}

# Data Structures

ValidationResult represents the result of a validation:

	type ValidationResult struct {
		Valid bool `json:"valid"`
		Errors []ValidationError `json:"errors,omitempty"`
	}

ValidationError provides details about validation errors:

	type ValidationError struct {
		Field string `json:"field"` // Field that failed validation
		Message string `json:"message"` // Error message
		Value interface{} `json:"value,omitempty"` // Value that caused the error
		Constraint string `json:"constraint,omitempty"` // Type of constraint violated
	}

# Error Handling

The library differentiates between validation errors (invalid data) and operational errors:

- Operational errors (file not found, malformed JSON, etc.) are returned as error
- Invalid data results in ValidationResult.Valid = false with details in ValidationResult.Errors

# Compatibility

This library is compatible with JSON Schema Draft 7 and supports all its versions Features:

- Basic type validation (string, number, integer, boolean, array, object, null)
- Numeric constraints (minimum, maximum, multipleOf, etc.)
- String constraints (minLength, maxLength, pattern, format)
- Array constraints (minItems, maxItems, uniqueItems, items, etc.)
- Object constraints (properties, required, additionalProperties, etc.)
- Conditional validation (if/then/else, allOf, anyOf, oneOf, not)
- References ($ref)
- Custom formats

# Dependencies

This library uses github.com/xeipuuv/gojsonschema for JSON Schema validation.

# Complete Examples

See the test files (*_test.go) for complete usage examples and test cases.

# License

[Specify license here]

# Contribution

[Specify contribution guidelines here]
*/
package valid
