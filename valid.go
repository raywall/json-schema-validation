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

// ValidationError representa um erro de validação detalhado para um campo específico.
// Eu projetei esta estrutura para fornecer feedback claro e estruturado para os clientes da API.
type ValidationError struct {
	Field      string      `json:"field"`                // O campo que falhou na validação.
	Message    string      `json:"message"`              // A mensagem de erro (pode ser personalizada).
	Value      interface{} `json:"value,omitempty"`      // O valor que causou o erro.
	Constraint string      `json:"constraint,omitempty"` // A restrição do schema que foi violada (ex: "minLength").
	Context    string      `json:"context,omitempty"`    // O contexto do erro, fornecido pela biblioteca de validação.
}

// ValidationResult representa o resultado completo de uma operação de validação.
type ValidationResult struct {
	Valid  bool              `json:"valid"`            // `true` se os dados forem válidos, `false` caso contrário.
	Errors []ValidationError `json:"errors,omitempty"` // Uma lista de erros de validação se Valid for `false`.
}

// ErrorResponse representa uma resposta de erro HTTP padrão que pode ser usada pelo middleware.
type ErrorResponse struct {
	Error   string            `json:"error"`             // Uma mensagem de erro geral.
	Details []ValidationError `json:"details,omitempty"` // Os detalhes específicos dos erros de validação.
}

// Validator é a estrutura principal que encapsula um schema JSON e fornece os métodos de validação.
// Cada instância de Validator está ligada a um único schema.
type Validator struct {
	schema       gojsonschema.JSONLoader
	customErrors map[string]map[string]string // Um mapa de mensagens de erro personalizadas extraídas do schema.
}

// New cria um novo Validator a partir de um arquivo de schema no sistema de arquivos.
// Esta é a forma recomendada de carregar schemas que estão armazenados junto com a aplicação.
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

	return NewFromBytes(schemaBytes)
}

// NewFromString cria um novo Validator a partir de uma string contendo o schema JSON.
// Útil para schemas que são gerados dinamicamente ou embutidos no código.
func NewFromString(schemaJSON string) (*Validator, error) {
	if strings.TrimSpace(schemaJSON) == "" {
		return nil, fmt.Errorf("schema não pode estar vazio")
	}
	return NewFromBytes([]byte(schemaJSON))
}

// NewFromBytes cria um novo Validator a partir de um slice de bytes do schema JSON.
// Este é o construtor base que os outros utilizam. Ele também aciona a extração de mensagens de erro personalizadas.
func NewFromBytes(schemaBytes []byte) (*Validator, error) {
	if len(schemaBytes) == 0 {
		return nil, fmt.Errorf("schema bytes não podem estar vazios")
	}

	// Parse the schema to extract custom error messages
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaObj); err != nil {
		return nil, fmt.Errorf("schema JSON inválido: %w", err)
	}

	// Extract custom error messages from schema
	customErrors := extractErrorMessages(schemaObj)

	schema := gojsonschema.NewBytesLoader(schemaBytes)

	return &Validator{
		schema:       schema,
		customErrors: customErrors,
	}, nil
}

// extractErrorMessages é uma função auxiliar que percorre o schema JSON para extrair
// mensagens de erro personalizadas definidas na propriedade `errorMessage`.
func extractErrorMessages(schema map[string]interface{}) map[string]map[string]string {
	errorMessages := make(map[string]map[string]string)

	if items, ok := schema["items"].(map[string]interface{}); ok {
		if props, ok := items["properties"].(map[string]interface{}); ok {
			for field, prop := range props {
				if propMap, ok := prop.(map[string]interface{}); ok {
					if errMsg, ok := propMap["errorMessage"].(map[string]interface{}); ok {
						fieldErrors := make(map[string]string)
						for key, msg := range errMsg {
							if msgStr, ok := msg.(string); ok {
								fieldErrors[key] = msgStr
							}
						}
						errorMessages[field] = fieldErrors
					}
				}
			}
		}

		// Extract required field messages
		if errMsg, ok := items["errorMessage"].(map[string]interface{}); ok {
			if requiredMsgs, ok := errMsg["required"].(map[string]interface{}); ok {
				for field, msg := range requiredMsgs {
					if msgStr, ok := msg.(string); ok {
						if _, exists := errorMessages[field]; !exists {
							errorMessages[field] = make(map[string]string)
						}
						errorMessages[field]["required"] = msgStr
					}
				}
			}
		}
	}

	return errorMessages
}

// ValidateRequest lê o corpo de uma requisição HTTP, valida-o contra o schema e retorna o resultado.
// Importante: eu projetei esta função para que o corpo da requisição possa ser lido novamente
// pelo handler subsequente, o que é um requisito comum em middlewares.
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

// ValidateBytes valida um slice de bytes contendo dados JSON contra o schema.
// Esta é a função de validação central da biblioteca.
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

// buildValidationResult constrói a estrutura ValidationResult a partir do resultado bruto
// da biblioteca gojsonschema, substituindo as mensagens de erro padrão pelas personalizadas, se disponíveis.
func (v *Validator) buildValidationResult(result *gojsonschema.Result) *ValidationResult {
	validationResult := &ValidationResult{
		Valid: result.Valid(),
	}

	if !result.Valid() {
		validationResult.Errors = make([]ValidationError, 0, len(result.Errors()))

		for _, err := range result.Errors() {
			field := strings.TrimPrefix(err.Field(), "(root).")
			if field == "(root)" {
				field = ""
			}

			// Try to get custom error message
			message := v.getCustomErrorMessage(field, err)

			validationErr := ValidationError{
				Field:      field,
				Message:    message,
				Constraint: err.Type(),
				Context:    err.Context().String(),
			}

			if err.Value() != nil {
				validationErr.Value = err.Value()
			}

			validationResult.Errors = append(validationResult.Errors, validationErr)
		}
	}

	return validationResult
}

// getCustomErrorMessage tenta encontrar uma mensagem de erro personalizada para um erro de validação específico.
// Ele procura por mensagens específicas para a restrição (ex: "required") e também por mensagens genéricas.
func (v *Validator) getCustomErrorMessage(field string, err gojsonschema.ResultError) string {
	// Split field path for nested properties
	fieldPath := strings.Split(field, ".")
	baseField := fieldPath[0]

	if fieldMessages, ok := v.customErrors[baseField]; ok {
		// Check for specific constraint message
		if msg, ok := fieldMessages[err.Type()]; ok {
			return msg
		}

		// Check for generic message
		if msg, ok := fieldMessages["_"]; ok {
			return msg
		}
	}

	// Fallback to default description
	return err.Description()
}

// ValidateString é um método de conveniência que valida uma string JSON contra o schema.
func (v *Validator) ValidateString(jsonString string) (*ValidationResult, error) {
	return v.ValidateBytes([]byte(jsonString))
}

// ValidateInterface serializa uma estrutura ou mapa Go para JSON e depois a valida contra o schema.
func (v *Validator) ValidateInterface(data interface{}) (*ValidationResult, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar dados para JSON: %w", err)
	}

	return v.ValidateBytes(jsonBytes)
}

// Middleware retorna um middleware HTTP com configurações padrão para validação automática.
// Por padrão, ele pula a validação para métodos como GET e DELETE e usa um manipulador de erros JSON padrão.
func (v *Validator) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return v.MiddlewareWithConfig(MiddlewareConfig{}, next)
}

// MiddlewareConfig define as configurações para o middleware de validação.
type MiddlewareConfig struct {
	// SkipMethods especifica uma lista de métodos HTTP que devem pular a validação.
	SkipMethods []string
	// ErrorHandler permite que você defina uma função personalizada para tratar os erros de validação.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, result *ValidationResult)
}

// MiddlewareWithConfig retorna um middleware HTTP com configurações personalizadas.
// Isso lhe dá controle total sobre quais métodos validar e como os erros são reportados ao cliente.
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

// defaultErrorHandler é o manipulador de erros padrão usado pelo middleware.
// Ele responde com um status 400 (Bad Request) e um corpo JSON contendo os detalhes do erro.
func (v *Validator) defaultErrorHandler(w http.ResponseWriter, r *http.Request, result *ValidationResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := ErrorResponse{
		Error:   "Dados de entrada inválidos",
		Details: result.Errors,
	}

	json.NewEncoder(w).Encode(response)
}

// MultiValidator gerencia uma coleção de validadores nomeados.
// Eu o criei para simplificar o gerenciamento de schemas em APIs complexas
// onde cada endpoint pode ter seu próprio schema de validação.
type MultiValidator struct {
	validators map[string]*Validator
}

// NewMultiValidator cria um novo gerenciador de múltiplos validadores.
func NewMultiValidator() *MultiValidator {
	return &MultiValidator{
		validators: make(map[string]*Validator),
	}
}

// Add adiciona um validador pré-construído ao gerenciador com uma chave específica.
func (mv *MultiValidator) Add(key string, validator *Validator) {
	mv.validators[key] = validator
}

// AddFromFile é um método de conveniência para carregar um schema de um arquivo e adicioná-lo ao gerenciador.
func (mv *MultiValidator) AddFromFile(key, schemaPath string) error {
	validator, err := New(schemaPath)
	if err != nil {
		return err
	}
	mv.Add(key, validator)
	return nil
}

// AddFromString é um método de conveniência para criar um validador a partir de uma string de schema e adicioná-lo.
func (mv *MultiValidator) AddFromString(key, schemaJSON string) error {
	validator, err := NewFromString(schemaJSON)
	if err != nil {
		return err
	}
	mv.Add(key, validator)
	return nil
}

// Get retorna um validador do gerenciador pela sua chave.
func (mv *MultiValidator) Get(key string) (*Validator, bool) {
	validator, exists := mv.validators[key]
	return validator, exists
}

// Remove remove um validador do gerenciador.
func (mv *MultiValidator) Remove(key string) {
	delete(mv.validators, key)
}

// Keys retorna uma lista de todas as chaves de validadores registrados.
func (mv *MultiValidator) Keys() []string {
	keys := make([]string, 0, len(mv.validators))
	for key := range mv.validators {
		keys = append(keys, key)
	}
	return keys
}

// Count retorna o número de validadores registrados no gerenciador.
func (mv *MultiValidator) Count() int {
	return len(mv.validators)
}
