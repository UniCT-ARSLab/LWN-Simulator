package types

import "testing"

func TestValidationErrorsExposeStructuredIssues(t *testing.T) {
	var errors ValidationErrors
	if errors.Error() != "validation failed" {
		t.Fatalf("unexpected empty validation message: %q", errors.Error())
	}
	errors.Add("device.name", "required", "name is required")
	errors.Add("device.class", "invalid_enum", "class is invalid")
	if errors.Error() != "device.name: name is required; device.class: class is invalid" {
		t.Fatalf("unexpected validation message: %q", errors.Error())
	}
}
