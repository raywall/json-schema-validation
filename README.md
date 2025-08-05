# JSON Schema Validation para Go

Olá! Eu sou Raywall Malheiros. Criei esta biblioteca para resolver um problema que encontrei muitas vezes no desenvolvimento de APIs em Go: a necessidade de uma forma declarativa, poderosa e simples de validar payloads JSON de entrada.

A abordagem desta biblioteca é usar o poder do **JSON Schema** para definir a estrutura e as restrições dos seus dados. Uma vez que o schema está definido, a biblioteca cuida de toda a lógica de validação, incluindo a integração perfeita com o servidor HTTP do Go.

## Principais Funcionalidades

- ✅ **Validação Robusta:** Utiliza a especificação JSON Schema Draft 7 para validar tipos, formatos, comprimentos, padrões e muito mais.
- 🚀 **Middleware para HTTP:** Inclui um middleware para `net/http` que valida automaticamente os corpos das requisições, simplificando imensamente seus handlers.
- 🎨 **Mensagens de Erro Personalizadas:** Defina mensagens de erro amigáveis diretamente no seu schema JSON, proporcionando um feedback melhor para os usuários da sua API.
- 📚 **Suporte a Múltiplos Schemas:** Gerencie facilmente uma coleção de validadores para diferentes endpoints ou versões da sua API com o `MultiValidator`.
- 🧩 **Flexibilidade:** Valide dados de múltiplas fontes: arquivos, strings, `[]byte` ou qualquer `interface{}` do Go.

## Instalação

```bash
go get [github.com/raywall/json-schema-validation](https://github.com/raywall/json-schema-validation)
```

## Guia Rápido

O uso é bastante direto. Primeiro, crie um `Validator` a partir do seu schema:

```go
package main

import (
    "fmt"
    "log"
    "[github.com/raywall/json-schema-validation](https://github.com/raywall/json-schema-validation)" // Supondo que este seja o caminho do seu módulo
)

func main() {
    // Crie um validador a partir de uma string de schema
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
        log.Fatalf("Erro ao criar validador: %v", err)
    }

    // Valide um payload JSON válido
    validData := `{"name": "João Silva", "email": "joao@exemplo.com"}`
    result, _ := validator.ValidateString(validData)
    fmt.Printf("Dados válidos: %v\n", result.Valid) // Saída: Dados válidos: true

    // Valide um payload JSON inválido
    invalidData := `{"name": "J"}`
    result, _ = validator.ValidateString(invalidData)
    fmt.Printf("Dados inválidos: %v\n", result.Valid) // Saída: Dados inválidos: false
    // Imprima os erros detalhados
    for _, vErr := range result.Errors {
        fmt.Printf(" -> Campo: '%s', Erro: '%s'\n", vErr.Field, vErr.Message)
    }
}
```

## Uso Avançado

### Middleware HTTP

Para mim, esta é uma das funcionalidades mais úteis. Em vez de chamar o validador manualmente em cada handler, você pode simplesmente envolver seu handler com o middleware.

```go
func userHandler(w http.ResponseWriter, r *http.Request) {
    // Se o código chegou aqui, a validação já passou.
    // Você pode processar a requisição com segurança.
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Dados do usuário processados com sucesso!"))
}

func main() {
    // ... criação do validator
    http.HandleFunc("/users", validator.Middleware(userHandler))
    log.Println("Servidor iniciado em :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Mensagens de Erro Personalizadas

Para melhorar a experiência do desenvolvedor que consome sua API, você pode definir mensagens de erro personalizadas diretamente no seu schema usando a propriedade `errorMessage`.

**Exemplo de Schema:**

```json
{
  "type": "object",
  "properties": {
    "email": {
      "type": "string",
      "format": "email",
      "errorMessage": {
        "format": "Por favor, forneça um endereço de e-mail válido."
      }
    },
    "password": {
      "type": "string",
      "minLength": 8,
      "errorMessage": {
        "minLength": "A senha deve ter pelo menos 8 caracteres.",
        "required": "O campo 'password' é obrigatório."
      }
    }
  },
  "required": ["email", "password"]
}
```

Quando a validação falhar, a `ValidationError` conterá a sua mensagem personalizada em vez da mensagem padrão da biblioteca.

### Gerenciando Múltiplos Schemas (`MultiValidator`)

Se sua aplicação tem múltiplos endpoints (ex: `/users`, `/products`, `/orders`), cada um com seu próprio schema, o `MultiValidator` simplifica o gerenciamento.

```go
// Crie um gerenciador
multiValidator := valid.NewMultiValidator()

// Adicione seus schemas
err := multiValidator.AddFromFile("user", "schemas/user.json")
if err != nil { /* ... */ }

err = multiValidator.AddFromFile("product", "schemas/product.json")
if err != nil { /* ... */ }

// Crie um handler que usa o validador correto
func createHandler(w http.ResponseWriter, r *http.Request) {
    // Determine qual validador usar (aqui, "user")
    userValidator, ok := multiValidator.Get("user")
    if !ok {
        http.Error(w, "Validador não encontrado", http.StatusInternalServerError)
        return
    }

    // Use o middleware do validador específico
    middleware := userValidator.Middleware(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Usuário criado com sucesso!"))
    })
    middleware(w, r)
}
```

---

Espero que esta documentação ajude a tornar sua biblioteca ainda mais profissional e fácil de usar!

Atenciosamente,

**Raywall Malheiros**
