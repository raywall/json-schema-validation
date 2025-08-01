/*
Package validator fornece funcionalidades de validação JSON Schema para requisições HTTP e dados JSON.

Esta biblioteca oferece uma interface simples e flexível para validar dados JSON contra schemas JSON Schema Draft 7,
com suporte especializado para aplicações web e APIs.

# Características Principais

- Validação de dados JSON contra JSON Schema Draft 7
- Middleware HTTP para validação automática de requisições
- Suporte a múltiplos validadores para diferentes endpoints
- Mensagens de erro detalhadas e estruturadas
- Integração fácil com aplicações Go existentes
- Validação de arquivos, strings, bytes e interfaces

# Uso Básico

Criar um validador a partir de um arquivo de schema:

	validator, err := validator.New("user-schema.json")
	if err != nil {
		log.Fatal(err)
	}

Criar um validador a partir de uma string JSON:

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

# Validação de Dados

Validar uma string JSON:

	result, err := validator.ValidateString(`{"name": "João", "email": "joao@exemplo.com"}`)
	if err != nil {
		log.Fatal(err)
	}

	if !result.Valid {
		for _, err := range result.Errors {
			fmt.Printf("Campo: %s, Erro: %s\n", err.Field, err.Message)
		}
	}

Validar uma requisição HTTP:

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
				"error": "Dados inválidos",
				"details": result.Errors,
			})
			return
		}

		// Processar dados válidos...
	}

# Middleware HTTP

Usar como middleware para validação automática:

	http.HandleFunc("/users", validator.Middleware(userHandler))

Middleware com configurações personalizadas:

	config := validator.MiddlewareConfig{
		SkipMethods: []string{"GET", "DELETE"},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, result *validator.ValidationResult) {
			// Handler personalizado de erro
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors": result.Errors,
			})
		},
	}

	http.HandleFunc("/users", validator.MiddlewareWithConfig(config, userHandler))

# Múltiplos Validadores

Para aplicações com múltiplos endpoints e schemas diferentes:

	multiValidator := validator.NewMultiValidator()

	// Adicionar validadores
	err := multiValidator.AddFromFile("user", "schemas/user.json")
	if err != nil {
		log.Fatal(err)
	}

	err = multiValidator.AddFromString("product", productSchema)
	if err != nil {
		log.Fatal(err)
	}

	// Usar validadores específicos
	userValidator, exists := multiValidator.Get("user")
	if exists {
		result, err := userValidator.ValidateString(jsonData)
		// ...
	}

# Estruturas de Dados

ValidationResult representa o resultado de uma validação:

	type ValidationResult struct {
		Valid  bool              `json:"valid"`
		Errors []ValidationError `json:"errors,omitempty"`
	}

ValidationError fornece detalhes sobre erros de validação:

	type ValidationError struct {
		Field       string      `json:"field"`        // Campo que falhou na validação
		Message     string      `json:"message"`      // Mensagem de erro
		Value       interface{} `json:"value,omitempty"` // Valor que causou o erro
		Constraint  string      `json:"constraint,omitempty"` // Tipo de restrição violada
	}

# Tratamento de Erros

A biblioteca diferencia entre erros de validação (dados inválidos) e erros operacionais:

- Erros operacionais (arquivo não encontrado, JSON malformado, etc.) são retornados como error
- Dados inválidos resultam em ValidationResult.Valid = false com detalhes em ValidationResult.Errors

# Compatibilidade

Esta biblioteca é compatível com JSON Schema Draft 7 e suporta todas as suas funcionalidades:

- Validação de tipos básicos (string, number, integer, boolean, array, object, null)
- Restrições numéricas (minimum, maximum, multipleOf, etc.)
- Restrições de string (minLength, maxLength, pattern, format)
- Restrições de array (minItems, maxItems, uniqueItems, items, etc.)
- Restrições de objeto (properties, required, additionalProperties, etc.)
- Validação condicional (if/then/else, allOf, anyOf, oneOf, not)
- Referências ($ref)
- Formatos customizados

# Dependências

Esta biblioteca utiliza github.com/xeipuuv/gojsonschema para a validação JSON Schema.

# Exemplos Completos

Ver os arquivos de teste (*_test.go) para exemplos completos de uso e casos de teste.

# Licença

[Especificar licença aqui]

# Contribuição

[Especificar diretrizes de contribuição aqui]
*/
package valid
