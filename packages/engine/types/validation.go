package types

import (
	"fmt"
	"strings"
)

type ValidationIssue struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationErrors struct {
	Issues []ValidationIssue `json:"fields"`
}

func (e *ValidationErrors) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "validation failed"
	}

	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
	}

	return strings.Join(parts, "; ")
}

func (e *ValidationErrors) Add(field string, code string, message string) {
	e.Issues = append(e.Issues, ValidationIssue{
		Field:   field,
		Code:    code,
		Message: message,
	})
}
