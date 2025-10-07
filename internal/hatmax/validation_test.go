package hatmax

import (
	"context"
	"testing"

	"github.com/adrianpk/hatmax/pkg/lib/hm"
)

// Mock Item struct for testing
type Item struct {
	Text string
	Done bool
}

func alwaysValidCreate(ctx context.Context, item Item) []hm.ValidationError {
	return nil
}

func alwaysInvalidCreate(ctx context.Context, item Item) []hm.ValidationError {
	return []hm.ValidationError{
		{Field: "text", Code: "test_error", Message: "Always invalid for testing"},
	}
}

func invalidOnlyForText(invalidText string) func(context.Context, Item) []hm.ValidationError {
	return func(ctx context.Context, item Item) []hm.ValidationError {
		if item.Text == invalidText {
			return []hm.ValidationError{
				{Field: "text", Code: "forbidden", Message: "This text is not allowed"},
			}
		}
		return nil
	}
}

func TestFunctionalValidationFlexibility(t *testing.T) {
	tests := []struct {
		name          string
		validateFunc  func(context.Context, Item) []hm.ValidationError
		item          Item
		expectErrors  bool
		expectedField string
		expectedCode  string
	}{
		{
			name:         "always valid validator",
			validateFunc: alwaysValidCreate,
			item:         Item{Text: "anything", Done: false},
			expectErrors: false,
		},
		{
			name:          "always invalid validator",
			validateFunc:  alwaysInvalidCreate,
			item:          Item{Text: "anything", Done: false},
			expectErrors:  true,
			expectedField: "text",
			expectedCode:  "test_error",
		},
		{
			name:         "conditional validator - valid text",
			validateFunc: invalidOnlyForText("forbidden"),
			item:         Item{Text: "allowed", Done: false},
			expectErrors: false,
		},
		{
			name:          "conditional validator - invalid text",
			validateFunc:  invalidOnlyForText("forbidden"),
			item:          Item{Text: "forbidden", Done: false},
			expectErrors:  true,
			expectedField: "text",
			expectedCode:  "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			errors := tt.validateFunc(ctx, tt.item)

			if tt.expectErrors {
				if len(errors) == 0 {
					t.Errorf("Expected validation errors, but got none")
					return
				}

				if errors[0].Field != tt.expectedField {
					t.Errorf("Expected field %q, got %q", tt.expectedField, errors[0].Field)
				}

				if errors[0].Code != tt.expectedCode {
					t.Errorf("Expected code %q, got %q", tt.expectedCode, errors[0].Code)
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("Expected no validation errors, but got: %v", errors)
				}
			}
		})
	}
}

func composeValidators(validators ...func(context.Context, Item) []hm.ValidationError) func(context.Context, Item) []hm.ValidationError {
	return func(ctx context.Context, item Item) []hm.ValidationError {
		var allErrors []hm.ValidationError
		for _, validator := range validators {
			errors := validator(ctx, item)
			allErrors = append(allErrors, errors...)
		}
		return allErrors
	}
}

func TestValidatorComposition(t *testing.T) {
	// Create composed validator
	composedValidator := composeValidators(
		invalidOnlyForText("bad"),
		func(ctx context.Context, item Item) []hm.ValidationError {
			if len(item.Text) < 3 {
				return []hm.ValidationError{
					{Field: "text", Code: "too_short", Message: "Text too short"},
				}
			}
			return nil
		},
	)

	tests := []struct {
		name         string
		item         Item
		expectErrors int
	}{
		{
			name:         "valid item",
			item:         Item{Text: "good text", Done: false},
			expectErrors: 0,
		},
		{
			name:         "short text",
			item:         Item{Text: "ab", Done: false},
			expectErrors: 1,
		},
		{
			name:         "bad text",
			item:         Item{Text: "bad", Done: false},
			expectErrors: 1,
		},
		{
			name:         "short and bad text",
			item:         Item{Text: "ba", Done: false},
			expectErrors: 1, // Only short text error since "ba" != "bad"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			errors := composedValidator(ctx, tt.item)

			if len(errors) != tt.expectErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectErrors, len(errors), errors)
			}
		})
	}
}
