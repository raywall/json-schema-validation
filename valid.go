// Package valid provides JSON Schema validation capabilities for HTTP requests and JSON data.
package valid

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ValidationError represents a detailed validation error
type ValidationError struct {
	Field      string      `json:"field"`
	Message    string      `json:"message"`
	Value      interface{} `json:"value,omitempty"`
	Constraint string      `json:"constraint,omitempty"`
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ErrorResponse represents the standard http error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details []ValidationError `json:"details,omitempty"`
}

// Validator encapsulates the Json Schema validator
type Validator struct {
	schema gojsonschema.JSONLoader
}

// New Creates a new validator from a Schema file
func New(schemaPath string) (*Validator, error) {
	schemaFile, err := os.Open(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir arquivo de schema '%s': %w", schemaPath, err)
	}
	defer schemaFile.Close()

	schemaBytes, err := io.ReadAll(schemaFile)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo de schema '%s': %w", schemaPath, err)
	}

	// Valida se o schema é um JSON válido
	var schemaObj interface{}
	if err := json.Unmarshal(schemaBytes, &schemaObj); err != nil {
		return nil, fmt.Errorf("schema inválido em '%s': %w", schemaPath, err)
	}

	schema := gojsonschema.NewBytesLoader(schemaBytes)

	return &Validator{
		schema: schema,
	}, nil
}

// NewFromString Creates a validator from a string JSON Schema
func NewFromString(schemaJSON string) (*Validator, error) {
	if strings.TrimSpace(schemaJSON) == "" {
		return nil, fmt.Errorf("schema não pode estar vazio")
	}

	// Validated if the Schema is a valid JSON
	var schemaObj interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaObj); err != nil {
		return nil, fmt.Errorf("schema JSON inválido: %w", err)
	}

	schema := gojsonschema.NewStringLoader(schemaJSON)

	return &Validator{
		schema: schema,
	}, nil
}

// NewFromBytes Creates a validator from bytes of a JSON Schema
func NewFromBytes(schemaBytes []byte) (*Validator, error) {
	if len(schemaBytes) == 0 {
		return nil, fmt.Errorf("schema bytes não podem estar vazios")
	}

	// Validated if the Schema is a valid JSON
	var schemaObj interface{}
	if err := json.Unmarshal(schemaBytes, &schemaObj); err != nil {
		return nil, fmt.Errorf("schema bytes inválidos: %w", err)
	}

	schema := gojsonschema.NewBytesLoader(schemaBytes)

	return &Validator{
		schema: schema,
	}, nil
}

// ValidateRequest validates an HTTP request against Schema
func (v *Validator) ValidateRequest(r *http.Request) (*ValidationResult, error) {
	if r == nil {
		return nil, fmt.Errorf("requisição não pode ser nil")
	}

	if r.Body == nil {
		return nil, fmt.Errorf("corpo da requisição não pode ser nil")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler corpo da requisição: %w", err)
	}

	// Allows to reuse the requisition body
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	return v.ValidateBytes(body)
}

// ValidateBytes validates JSON bytes against schema
func (v *Validator) ValidateBytes(jsonData []byte) (*ValidationResult, error) {
	if len(jsonData) == 0 {
		return nil, fmt.Errorf("dados JSON não podem estar vazios")
	}

	// Validates if it is valid JSON before validating the schema
	var jsonObj interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{
					Field:      "root",
					Message:    fmt.Sprintf("JSON inválido: %s", err.Error()),
					Constraint: "format",
				},
			},
		}, nil
	}

	document := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(v.schema, document)
	if err != nil {
		return nil, fmt.Errorf("erro durante validação do schema: %w", err)
	}

	return v.buildValidationResult(result), nil
}

// ValidateString validates a JSON string against the schema
func (v *Validator) ValidateString(jsonString string) (*ValidationResult, error) {
	return v.ValidateBytes([]byte(jsonString))
}

// ValidateInterface validates an interface{} against the schema
func (v *Validator) ValidateInterface(data interface{}) (*ValidationResult, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar dados para JSON: %w", err)
	}

	return v.ValidateBytes(jsonBytes)
}

// buildValidationResult builds the validation result from the gojsonschema result
func (v *Validator) buildValidationResult(result *gojsonschema.Result) *ValidationResult {
	validationResult := &ValidationResult{
		Valid: result.Valid(),
	}

	if !result.Valid() {
		validationResult.Errors = make([]ValidationError, 0, len(result.Errors()))

		for _, err := range result.Errors() {
			validationErr := ValidationError{
				Field:      err.Field(),
				Message:    err.Description(),
				Constraint: err.Type(),
			}

			// Tenta extrair o valor que causou o erro
			if err.Value() != nil {
				validationErr.Value = err.Value()
			}

			validationResult.Errors = append(validationResult.Errors, validationErr)
		}
	}

	return validationResult
}

// Middleware returns an HTTP middleware for automatic validation
func (v *Validator) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return v.MiddlewareWithConfig(MiddlewareConfig{}, next)
}

// MiddlewareConfig settings for the middleware
type MiddlewareConfig struct {
	// SkipMethods HTTP methods that should skip validation (default: GET, DELETE, HEAD)
	SkipMethods []string
	// ErrorHandler custom function to handle validation errors
	ErrorHandler func(w http.ResponseWriter, r *http.Request, result *ValidationResult)
}

// MiddlewareWithConfig returns an HTTP middleware with custom settings
func (v *Validator) MiddlewareWithConfig(config MiddlewareConfig, next http.HandlerFunc) http.HandlerFunc {
	// Default methods that skip validation
	if len(config.SkipMethods) == 0 {
		config.SkipMethods = []string{"GET", "DELETE", "HEAD", "OPTIONS"}
	}

	// Standard error handler
	if config.ErrorHandler == nil {
		config.ErrorHandler = v.defaultErrorHandler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Checks whether to skip validation for this method
		for _, method := range config.SkipMethods {
			if r.Method == method {
				next(w, r)
				return
			}
		}

		validation, err := v.ValidateRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro interno de validação: %s", err.Error()),
				http.StatusInternalServerError)
			return
		}

		if !validation.Valid {
			config.ErrorHandler(w, r, validation)
			return
		}

		next(w, r)
	}
}

// defaultErrorHandler is the default error handler for the middleware
func (v *Validator) defaultErrorHandler(w http.ResponseWriter, r *http.Request, result *ValidationResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := ErrorResponse{
		Error:   "Dados de entrada inválidos",
		Details: result.Errors,
	}

	json.NewEncoder(w).Encode(response)
}

// MultiValidator manages multiple validators
type MultiValidator struct {
	validators map[string]*Validator
}

// NewMultiValidator creates a new multiple validator manager
func NewMultiValidator() *MultiValidator {
	return &MultiValidator{
		validators: make(map[string]*Validator),
	}
}

// Add adds a validator with a specific key
func (mv *MultiValidator) Add(key string, validator *Validator) {
	mv.validators[key] = validator
}

// AddFromFile add a validator from a file
func (mv *MultiValidator) AddFromFile(key, schemaPath string) error {
	validator, err := New(schemaPath)
	if err != nil {
		return err
	}
	mv.Add(key, validator)
	return nil
}

// AddFromString adds a validator from a string
func (mv *MultiValidator) AddFromString(key, schemaJSON string) error {
	validator, err := NewFromString(schemaJSON)
	if err != nil {
		return err
	}
	mv.Add(key, validator)
	return nil
}

// Get returns a validator by key
func (mv *MultiValidator) Get(key string) (*Validator, bool) {
	validator, exists := mv.validators[key]
	return validator, exists
}

// Remove removes a validator
func (mv *MultiValidator) Remove(key string) {
	delete(mv.validators, key)
}

// Keys returns all validator keys
func (mv *MultiValidator) Keys() []string {
	keys := make([]string, 0, len(mv.validators))
	for key := range mv.validators {
		keys = append(keys, key)
	}
	return keys
}

// Count returns the number of registered validators
func (mv *MultiValidator) Count() int {
	return len(mv.validators)
}
