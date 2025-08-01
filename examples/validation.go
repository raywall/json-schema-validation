package main

import (
	"encoding/json"
	"fmt"
	"log"

	valid "github.com/raywall/json-schema-validation"
)

var (
	validator *valid.Validator
	err       error
)

type Item struct {
	ID        string `json:"id"`       // this is a string with UUID format
	Category  string `json:"category"` // here will be accepted customer or supplier only
	Activated bool   `json:"activated"`
}

func init() {
	// Define JSON schema for ListItem
	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"id": {
				"type": "string",
				"format": "uuid",
				"pattern": "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
			},
			"category": {
				"type": "string",
				"enum": ["customer", "supplier"],
				"minLength": 1
			},
			"activated": {
				"type": "boolean"
			}
		},
		"required": ["id", "category", "activated"],
		"additionalProperties": false
	}`

	// validator initialize
	validator, err = valid.NewFromString(schemaJSON)
	if err != nil {
		log.Fatalf("failed to create validator, %v", err)
	}
}

func main() {
	// valid data
	data := Item{
		ID:        "f57ef656-bb28-4464-89e4-f3815aa7cdc9",
		Category:  "customer",
		Activated: false,
	}

	if err := data.Validate(); err != nil {
		fmt.Println("Error validating data:", err)
		return
	}
	fmt.Println("Data is valid")

	// invalid data, error expected
	data = Item{
		ID:        "teste",
		Category:  "customer",
		Activated: false,
	}

	if err := data.Validate(); err != nil {
		fmt.Println("Error validating data:", err)
		return
	}
}

func (i *Item) Validate() error {
	jsonData, err := json.Marshal(*i)
	if err != nil {
		return fmt.Errorf("failed to marshal item for validation, %v", err)
	}

	result, err := validator.ValidateString(string(jsonData))
	if err != nil {
		return fmt.Errorf("validation error, %v", err)
	}

	if !result.Valid {
		var errorMessages []string
		for _, err := range result.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("Field: %s, Error: %s", err.Field, err.Message))
		}
		return fmt.Errorf("validation failed: %v", errorMessages)
	}

	return nil
}
