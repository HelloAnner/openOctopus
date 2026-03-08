package errors

import "fmt"

type Category string

const (
	CategorySyntax    Category = "syntax"
	CategorySchema    Category = "schema"
	CategoryReference Category = "reference"
	CategorySecurity  Category = "security"
	CategoryPolicy    Category = "policy"
)

type ConfigError struct {
	Code       string
	Category   Category
	Path       string
	Message    string
	Suggestion string
	RuleID     string
}

func (errorItem ConfigError) Error() string {
	return fmt.Sprintf("%s %s: %s", errorItem.Category, errorItem.Path, errorItem.Message)
}
