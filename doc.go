// Copyright 2025 Raywall Malheiros
//
/*
Package valid provides comprehensive JSON Schema validation capabilities for Go applications.

I created this library to offer a simple and powerful interface for validating JSON data
against JSON Schema specifications, with a special focus on the needs of web applications and APIs.
My goal was to create a tool that is both easy to integrate and flexible enough to handle
complex validation scenarios.

# Principais Funcionalidades

- **Validação de JSON Schema:** Suporte robusto para JSON Schema Draft 7.
- **Middleware HTTP:** Middleware para Go que valida automaticamente os corpos das requisições HTTP,
  reduzindo o código repetitivo nos seus handlers.
- **Suporte a Múltiplos Validadores:** Gerencie facilmente múltiplos schemas para diferentes
  endpoints ou versões de API usando o `MultiValidator`.
- **Mensagens de Erro Estruturadas e Personalizáveis:** Retorna erros de validação detalhados e permite
  a personalização de mensagens de erro diretamente no seu arquivo de schema.
- **Integração Flexível:** Valide dados de arquivos, strings, slices de bytes ou de qualquer
  interface Go (`interface{}`).

# Uso Básico

Primeiro, crie um validador a partir de um arquivo de schema:

	validator, err := valid.New("esquema-usuario.json")
	if err != nil {
		log.Fatal(err)
	}

Ou crie a partir de uma string JSON:

	schemaJSON := `{
		"type": "object",
		"properties": {
			"name": {"type": "string", "minLength": 2},
			"email": {"type": "string", "format": "email"}
		},
		"required": ["name", "email"]
	}`

	validator, err := valid.NewFromString(schemaJSON)
	if err != nil {
		log.Fatal(err)
	}

# Validando Dados

Para validar uma string JSON:

	result, err := validator.ValidateString(`{"name": "João", "email": "joao@exemplo.com"}`)
	if err != nil {
		log.Fatal(err)
	}

	if !result.Valid {
		for _, vErr := range result.Errors {
			fmt.Printf("Campo: %s, Erro: %s\n", vErr.Field, vErr.Message)
		}
	}

# Middleware HTTP

Eu projetei o middleware para ser extremamente fácil de usar. Para validação automática,
envolva seu `http.HandlerFunc`:

	http.HandleFunc("/users", validator.Middleware(meuUserHandler))

Você também pode configurar o middleware, por exemplo, para pular a validação de certos
métodos HTTP ou para usar um manipulador de erros customizado:

	config := valid.MiddlewareConfig{
		SkipMethods: []string{"GET", "DELETE"},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, result *valid.ValidationResult) {
			// Lógica customizada para responder a erros de validação.
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{"valid": false, "validation_errors": result.Errors})
		},
	}
	http.HandleFunc("/users", validator.MiddlewareWithConfig(config, meuUserHandler))

# Múltiplos Validadores

Para aplicações maiores com múltiplos schemas, eu criei o `MultiValidator`:

	multiValidator := valid.NewMultiValidator()
	multiValidator.AddFromFile("user", "schemas/user.json")
	multiValidator.AddFromString("product", productSchemaJSON)

	// Em seu handler, recupere o validador apropriado:
	userValidator, ok := multiValidator.Get("user")
	if ok {
		// use userValidator...
	}
*/
package valid
