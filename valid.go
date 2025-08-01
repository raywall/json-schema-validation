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

// ValidationError representa um erro de validação detalhado
type ValidationError struct {
	Field      string      `json:"field"`
	Message    string      `json:"message"`
	Value      interface{} `json:"value,omitempty"`
	Constraint string      `json:"constraint,omitempty"`
}

// ValidationResult representa o resultado de uma validação
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ErrorResponse representa a resposta de erro HTTP padrão
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details []ValidationError `json:"details,omitempty"`
}

// Validator encapsula o validador de JSON Schema
type Validator struct {
	schema gojsonschema.JSONLoader
}

// New cria um novo validador a partir de um arquivo de schema
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

// NewFromString cria um validador a partir de uma string JSON Schema
func NewFromString(schemaJSON string) (*Validator, error) {
	if strings.TrimSpace(schemaJSON) == "" {
		return nil, fmt.Errorf("schema não pode estar vazio")
	}

	// Valida se o schema é um JSON válido
	var schemaObj interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaObj); err != nil {
		return nil, fmt.Errorf("schema JSON inválido: %w", err)
	}

	schema := gojsonschema.NewStringLoader(schemaJSON)

	return &Validator{
		schema: schema,
	}, nil
}

// NewFromBytes cria um validador a partir de bytes de um JSON Schema
func NewFromBytes(schemaBytes []byte) (*Validator, error) {
	if len(schemaBytes) == 0 {
		return nil, fmt.Errorf("schema bytes não podem estar vazios")
	}

	// Valida se o schema é um JSON válido
	var schemaObj interface{}
	if err := json.Unmarshal(schemaBytes, &schemaObj); err != nil {
		return nil, fmt.Errorf("schema bytes inválidos: %w", err)
	}

	schema := gojsonschema.NewBytesLoader(schemaBytes)

	return &Validator{
		schema: schema,
	}, nil
}

// ValidateRequest valida uma requisição HTTP contra o schema
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

	// Permite reutilizar o body da requisição
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	return v.ValidateBytes(body)
}

// ValidateBytes valida bytes JSON contra o schema
func (v *Validator) ValidateBytes(jsonData []byte) (*ValidationResult, error) {
	if len(jsonData) == 0 {
		return nil, fmt.Errorf("dados JSON não podem estar vazios")
	}

	// Valida se é um JSON válido antes de validar o schema
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

// ValidateString valida uma string JSON contra o schema
func (v *Validator) ValidateString(jsonString string) (*ValidationResult, error) {
	return v.ValidateBytes([]byte(jsonString))
}

// ValidateInterface valida uma interface{} contra o schema
func (v *Validator) ValidateInterface(data interface{}) (*ValidationResult, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar dados para JSON: %w", err)
	}

	return v.ValidateBytes(jsonBytes)
}

// buildValidationResult constrói o resultado da validação a partir do resultado do gojsonschema
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

// Middleware retorna um middleware HTTP para validação automática
func (v *Validator) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return v.MiddlewareWithConfig(MiddlewareConfig{}, next)
}

// MiddlewareConfig configurações para o middleware
type MiddlewareConfig struct {
	// SkipMethods métodos HTTP que devem pular a validação (padrão: GET, DELETE, HEAD)
	SkipMethods []string
	// ErrorHandler função personalizada para tratar erros de validação
	ErrorHandler func(w http.ResponseWriter, r *http.Request, result *ValidationResult)
}

// MiddlewareWithConfig retorna um middleware HTTP com configurações personalizadas
func (v *Validator) MiddlewareWithConfig(config MiddlewareConfig, next http.HandlerFunc) http.HandlerFunc {
	// Métodos padrão que pulam validação
	if len(config.SkipMethods) == 0 {
		config.SkipMethods = []string{"GET", "DELETE", "HEAD", "OPTIONS"}
	}

	// Handler de erro padrão
	if config.ErrorHandler == nil {
		config.ErrorHandler = v.defaultErrorHandler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica se deve pular a validação para este método
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

// defaultErrorHandler é o handler de erro padrão para o middleware
func (v *Validator) defaultErrorHandler(w http.ResponseWriter, r *http.Request, result *ValidationResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := ErrorResponse{
		Error:   "Dados de entrada inválidos",
		Details: result.Errors,
	}

	json.NewEncoder(w).Encode(response)
}

// MultiValidator gerencia múltiplos validadores
type MultiValidator struct {
	validators map[string]*Validator
}

// NewMultiValidator cria um novo gerenciador de múltiplos validadores
func NewMultiValidator() *MultiValidator {
	return &MultiValidator{
		validators: make(map[string]*Validator),
	}
}

// Add adiciona um validador com uma chave específica
func (mv *MultiValidator) Add(key string, validator *Validator) {
	mv.validators[key] = validator
}

// AddFromFile adiciona um validador a partir de um arquivo
func (mv *MultiValidator) AddFromFile(key, schemaPath string) error {
	validator, err := New(schemaPath)
	if err != nil {
		return err
	}
	mv.Add(key, validator)
	return nil
}

// AddFromString adiciona um validador a partir de uma string
func (mv *MultiValidator) AddFromString(key, schemaJSON string) error {
	validator, err := NewFromString(schemaJSON)
	if err != nil {
		return err
	}
	mv.Add(key, validator)
	return nil
}

// Get retorna um validador pela chave
func (mv *MultiValidator) Get(key string) (*Validator, bool) {
	validator, exists := mv.validators[key]
	return validator, exists
}

// Remove remove um validador
func (mv *MultiValidator) Remove(key string) {
	delete(mv.validators, key)
}

// Keys retorna todas as chaves dos validadores
func (mv *MultiValidator) Keys() []string {
	keys := make([]string, 0, len(mv.validators))
	for key := range mv.validators {
		keys = append(keys, key)
	}
	return keys
}

// Count retorna o número de validadores registrados
func (mv *MultiValidator) Count() int {
	return len(mv.validators)
}
