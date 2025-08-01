package valid

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const testSchema = `{
	"type": "object",
	"properties": {
		"name": {
			"type": "string",
			"minLength": 2,
			"maxLength": 50
		},
		"email": {
			"type": "string",
			"format": "email"
		},
		"age": {
			"type": "integer",
			"minimum": 0,
			"maximum": 120
		},
		"address": {
			"type": "object",
			"properties": {
				"street": {"type": "string"},
				"city": {"type": "string"},
				"zipCode": {"type": "string", "pattern": "^[0-9]{5}-?[0-9]{3}$"}
			},
			"required": ["street", "city"]
		}
	},
	"required": ["name", "email"]
}`

func TestNewFromString(t *testing.T) {
	tests := []struct {
		name        string
		schema      string
		expectError bool
	}{
		{
			name:        "valid schema",
			schema:      testSchema,
			expectError: false,
		},
		{
			name:        "empty schema",
			schema:      "",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			schema:      `{"type": "object"`,
			expectError: true,
		},
		{
			name:        "whitespace only",
			schema:      "   ",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewFromString(tt.schema)
			if tt.expectError {
				if err == nil {
					t.Error("esperava erro, mas não recebeu nenhum")
				}
				if validator != nil {
					t.Error("esperava validator nil quando há erro")
				}
			} else {
				if err != nil {
					t.Errorf("não esperava erro, mas recebeu: %v", err)
				}
				if validator == nil {
					t.Error("esperava validator válido")
				}
			}
		})
	}
}

func TestNewFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		schema      []byte
		expectError bool
	}{
		{
			name:        "valid schema bytes",
			schema:      []byte(testSchema),
			expectError: false,
		},
		{
			name:        "empty bytes",
			schema:      []byte{},
			expectError: true,
		},
		{
			name:        "nil bytes",
			schema:      nil,
			expectError: true,
		},
		{
			name:        "invalid JSON bytes",
			schema:      []byte(`{"type": "object"`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewFromBytes(tt.schema)
			if tt.expectError {
				if err == nil {
					t.Error("esperava erro, mas não recebeu nenhum")
				}
			} else {
				if err != nil {
					t.Errorf("não esperava erro, mas recebeu: %v", err)
				}
				if validator == nil {
					t.Error("esperava validator válido")
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	// Cria arquivo temporário para teste
	tmpFile, err := os.CreateTemp("", "test-schema-*.json")
	if err != nil {
		t.Fatalf("erro ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Escreve schema no arquivo
	if _, err := tmpFile.WriteString(testSchema); err != nil {
		t.Fatalf("erro ao escrever no arquivo temporário: %v", err)
	}
	tmpFile.Close()

	validator, err := New(tmpFile.Name())
	if err != nil {
		t.Errorf("não esperava erro, mas recebeu: %v", err)
	}
	if validator == nil {
		t.Error("esperava validator válido")
	}

	// Teste com arquivo inexistente
	_, err = New("arquivo-inexistente.json")
	if err == nil {
		t.Error("esperava erro para arquivo inexistente")
	}
}

func TestValidateString(t *testing.T) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		t.Fatalf("erro ao criar validator: %v", err)
	}

	tests := []struct {
		name        string
		jsonData    string
		expectValid bool
		expectError bool
	}{
		{
			name: "valid data",
			jsonData: `{
				"name": "João Silva",
				"email": "joao@exemplo.com",
				"age": 30,
				"address": {
					"street": "Rua das Flores, 123",
					"city": "São Paulo",
					"zipCode": "01234-567"
				}
			}`,
			expectValid: true,
			expectError: false,
		},
		{
			name: "minimal valid data",
			jsonData: `{
				"name": "Ana",
				"email": "ana@test.com"
			}`,
			expectValid: true,
			expectError: false,
		},
		{
			name: "missing required field",
			jsonData: `{
				"name": "João Silva"
			}`,
			expectValid: false,
			expectError: false,
		},
		{
			name: "invalid email format",
			jsonData: `{
				"name": "João Silva",
				"email": "email-inválido"
			}`,
			expectValid: false,
			expectError: false,
		},
		{
			name: "name too short",
			jsonData: `{
				"name": "J",
				"email": "j@example.com"
			}`,
			expectValid: false,
			expectError: false,
		},
		{
			name: "negative age",
			jsonData: `{
				"name": "João Silva",
				"email": "joao@exemplo.com",
				"age": -5
			}`,
			expectValid: false,
			expectError: false,
		},
		{
			name: "invalid zipcode format",
			jsonData: `{
				"name": "João Silva",
				"email": "joao@exemplo.com",
				"address": {
					"street": "Rua das Flores, 123",
					"city": "São Paulo",
					"zipCode": "123"
				}
			}`,
			expectValid: false,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"name": "João"`,
			expectValid: false,
			expectError: false,
		},
		{
			name:        "empty string",
			jsonData:    "",
			expectValid: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateString(tt.jsonData)

			if tt.expectError {
				if err == nil {
					t.Error("esperava erro, mas não recebeu nenhum")
				}
				return
			}

			if err != nil {
				t.Errorf("não esperava erro, mas recebeu: %v", err)
				return
			}

			if result == nil {
				t.Error("resultado não deveria ser nil")
				return
			}

			if result.Valid != tt.expectValid {
				t.Errorf("esperava valid=%v, mas recebeu valid=%v", tt.expectValid, result.Valid)
				if !result.Valid && len(result.Errors) > 0 {
					t.Logf("Erros de validação: %+v", result.Errors)
				}
			}

			if !result.Valid && len(result.Errors) == 0 {
				t.Error("dados inválidos deveriam ter erros detalhados")
			}
		})
	}
}

func TestValidateBytes(t *testing.T) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		t.Fatalf("erro ao criar validator: %v", err)
	}

	validJSON := []byte(`{"name": "Test", "email": "test@example.com"}`)
	result, err := validator.ValidateBytes(validJSON)
	if err != nil {
		t.Errorf("não esperava erro: %v", err)
	}
	if !result.Valid {
		t.Error("esperava dados válidos")
	}

	// Teste com bytes vazios
	_, err = validator.ValidateBytes([]byte{})
	if err == nil {
		t.Error("esperava erro para bytes vazios")
	}
}

func TestValidateInterface(t *testing.T) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		t.Fatalf("erro ao criar validator: %v", err)
	}

	data := map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
		"age":   25,
	}

	result, err := validator.ValidateInterface(data)
	if err != nil {
		t.Errorf("não esperava erro: %v", err)
	}
	if !result.Valid {
		t.Errorf("esperava dados válidos, mas recebeu erros: %+v", result.Errors)
	}

	// Teste com dados inválidos
	invalidData := map[string]interface{}{
		"name":  "T", // muito curto
		"email": "invalid-email",
	}

	result, err = validator.ValidateInterface(invalidData)
	if err != nil {
		t.Errorf("não esperava erro: %v", err)
	}
	if result.Valid {
		t.Error("esperava dados inválidos")
	}
	if len(result.Errors) == 0 {
		t.Error("esperava erros de validação")
	}
}

func TestValidateRequest(t *testing.T) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		t.Fatalf("erro ao criar validator: %v", err)
	}

	// Teste com requisição válida
	validJSON := `{"name": "Test User", "email": "test@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(validJSON))
	req.Header.Set("Content-Type", "application/json")

	result, err := validator.ValidateRequest(req)
	if err != nil {
		t.Errorf("não esperava erro: %v", err)
	}
	if !result.Valid {
		t.Errorf("esperava dados válidos, mas recebeu erros: %+v", result.Errors)
	}

	// Verifica se o body pode ser lido novamente
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("erro ao ler body novamente: %v", err)
	}
	if string(body) != validJSON {
		t.Error("body da requisição foi modificado")
	}

	// Teste com requisição nil
	_, err = validator.ValidateRequest(nil)
	if err == nil {
		t.Error("esperava erro para requisição nil")
	}

	// Teste com body nil
	reqNilBody := &http.Request{}
	_, err = validator.ValidateRequest(reqNilBody)
	if err == nil {
		t.Error("esperava erro para body nil")
	}
}

func TestMiddleware(t *testing.T) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		t.Fatalf("erro ao criar validator: %v", err)
	}

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	middleware := validator.Middleware(handler)

	// Teste com GET (deve pular validação)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handlerCalled = false
	middleware(w, req)

	if !handlerCalled {
		t.Error("handler deveria ter sido chamado para GET")
	}
	if w.Code != http.StatusOK {
		t.Errorf("esperava status 200, recebeu %d", w.Code)
	}

	// Teste com POST válido
	validJSON := `{"name": "Test User", "email": "test@example.com"}`
	req = httptest.NewRequest("POST", "/test", strings.NewReader(validJSON))
	w = httptest.NewRecorder()

	handlerCalled = false
	middleware(w, req)

	if !handlerCalled {
		t.Error("handler deveria ter sido chamado para dados válidos")
	}
	if w.Code != http.StatusOK {
		t.Errorf("esperava status 200, recebeu %d", w.Code)
	}

	// Teste com POST inválido
	invalidJSON := `{"name": "T"}` // nome muito curto, email ausente
	req = httptest.NewRequest("POST", "/test", strings.NewReader(invalidJSON))
	w = httptest.NewRecorder()

	handlerCalled = false
	middleware(w, req)

	if handlerCalled {
		t.Error("handler não deveria ter sido chamado para dados inválidos")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava status 400, recebeu %d", w.Code)
	}

	// Verifica se a resposta de erro está no formato correto
	var errorResponse ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errorResponse)
	if err != nil {
		t.Errorf("erro ao decodificar resposta de erro: %v", err)
	}
	if errorResponse.Error == "" {
		t.Error("resposta de erro deveria ter mensagem")
	}
	if len(errorResponse.Details) == 0 {
		t.Error("resposta de erro deveria ter detalhes")
	}
}

func TestMiddlewareWithConfig(t *testing.T) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		t.Fatalf("erro ao criar validator: %v", err)
	}

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	customErrorHandlerCalled := false
	config := MiddlewareConfig{
		SkipMethods: []string{"GET", "POST"}, // Pula também POST
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, result *ValidationResult) {
			customErrorHandlerCalled = true
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors":  result.Errors,
			})
		},
	}

	middleware := validator.MiddlewareWithConfig(config, handler)

	// Teste POST (deve ser pulado devido à configuração)
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"invalid": "data"}`))
	w := httptest.NewRecorder()

	handlerCalled = false
	middleware(w, req)

	if !handlerCalled {
		t.Error("handler deveria ter sido chamado para POST (método pulado)")
	}

	// Teste PUT (deve validar e usar error handler customizado)
	req = httptest.NewRequest("PUT", "/test", strings.NewReader(`{"name": "T"}`))
	w = httptest.NewRecorder()

	handlerCalled = false
	customErrorHandlerCalled = false
	middleware(w, req)

	if handlerCalled {
		t.Error("handler não deveria ter sido chamado para dados inválidos")
	}
	if !customErrorHandlerCalled {
		t.Error("error handler customizado deveria ter sido chamado")
	}
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("esperava status 422, recebeu %d", w.Code)
	}
}

func TestMultiValidator(t *testing.T) {
	mv := NewMultiValidator()

	if mv.Count() != 0 {
		t.Error("MultiValidator deveria iniciar vazio")
	}

	// Teste AddFromString
	err := mv.AddFromString("user", testSchema)
	if err != nil {
		t.Errorf("erro ao adicionar validator: %v", err)
	}

	if mv.Count() != 1 {
		t.Error("MultiValidator deveria ter 1 validator após adicionar")
	}

	// Teste Get
	validator, exists := mv.Get("user")
	if !exists {
		t.Error("validator 'user' deveria existir")
	}
	if validator == nil {
		t.Error("validator não deveria ser nil")
	}

	// Teste com chave inexistente
	_, exists = mv.Get("inexistente")
	if exists {
		t.Error("validator inexistente não deveria existir")
	}

	// Teste Keys
	keys := mv.Keys()
	if len(keys) != 1 {
		t.Errorf("esperava 1 chave, recebeu %d", len(keys))
	}
	if keys[0] != "user" {
		t.Errorf("esperava chave 'user', recebeu '%s'", keys[0])
	}

	// Adiciona outro validator
	simpleSchema := `{"type": "string", "minLength": 1}`
	err = mv.AddFromString("simple", simpleSchema)
	if err != nil {
		t.Errorf("erro ao adicionar segundo validator: %v", err)
	}

	if mv.Count() != 2 {
		t.Error("MultiValidator deveria ter 2 validators")
	}

	// Teste Remove
	mv.Remove("user")
	if mv.Count() != 1 {
		t.Error("MultiValidator deveria ter 1 validator após remoção")
	}

	_, exists = mv.Get("user")
	if exists {
		t.Error("validator removido não deveria existir")
	}

	// Teste AddFromFile com arquivo inexistente
	err = mv.AddFromFile("test", "arquivo-inexistente.json")
	if err == nil {
		t.Error("esperava erro para arquivo inexistente")
	}
}

func TestValidationErrorStructure(t *testing.T) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		t.Fatalf("erro ao criar validator: %v", err)
	}

	invalidJSON := `{
		"name": "T",
		"email": "invalid-email",
		"age": -5
	}`

	result, err := validator.ValidateString(invalidJSON)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if result.Valid {
		t.Error("esperava dados inválidos")
	}

	if len(result.Errors) == 0 {
		t.Error("esperava erros de validação")
	}

	// Verifica estrutura dos erros
	for _, validationErr := range result.Errors {
		if validationErr.Message == "" {
			t.Error("erro deveria ter mensagem")
		}
		if validationErr.Constraint == "" {
			t.Error("erro deveria ter constraint")
		}
		// Field pode estar vazio para erros globais
		// Value pode ser nil para alguns tipos de erro
	}

	// Testa JSON malformado
	result, err = validator.ValidateString(`{"name": "test"`)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if result.Valid {
		t.Error("JSON malformado deveria ser inválido")
	}

	if len(result.Errors) != 1 {
		t.Errorf("esperava 1 erro para JSON malformado, recebeu %d", len(result.Errors))
	}

	if result.Errors[0].Field != "root" {
		t.Error("erro de JSON malformado deveria ter field 'root'")
	}
}

// Benchmarks
func BenchmarkValidateString(b *testing.B) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		b.Fatalf("erro ao criar validator: %v", err)
	}

	validJSON := `{
		"name": "João Silva",
		"email": "joao@exemplo.com",
		"age": 30,
		"address": {
			"street": "Rua das Flores, 123",
			"city": "São Paulo",
			"zipCode": "01234-567"
		}
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateString(validJSON)
		if err != nil {
			b.Fatalf("erro durante benchmark: %v", err)
		}
	}
}

func BenchmarkValidateBytes(b *testing.B) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		b.Fatalf("erro ao criar validator: %v", err)
	}

	validJSON := []byte(`{
		"name": "João Silva",
		"email": "joao@exemplo.com",
		"age": 30
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateBytes(validJSON)
		if err != nil {
			b.Fatalf("erro durante benchmark: %v", err)
		}
	}
}

func BenchmarkMiddleware(b *testing.B) {
	validator, err := NewFromString(testSchema)
	if err != nil {
		b.Fatalf("erro ao criar validator: %v", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	middleware := validator.Middleware(handler)
	validJSON := `{"name": "Test User", "email": "test@example.com"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader(validJSON))
		w := httptest.NewRecorder()
		middleware(w, req)
	}
}
