package validation

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidationErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  ValidationError
		want string
	}{
		{
			name: "error with field",
			err:  ValidationError{Field: "email", Message: "is required"},
			want: "email: is required",
		},
		{
			name: "error without field",
			err:  ValidationError{Field: "", Message: "validation failed"},
			want: "validation failed",
		},
		{
			name: "error with empty message",
			err:  ValidationError{Field: "name", Message: ""},
			want: "name: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationErrorsError(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		want   string
	}{
		{
			name:   "no errors",
			errors: ValidationErrors{},
			want:   "",
		},
		{
			name: "single error",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
			},
			want: "email: is required",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
				{Field: "password", Message: "is too short"},
			},
			want: "email: is required; password: is too short",
		},
		{
			name: "errors with and without fields",
			errors: ValidationErrors{
				{Field: "email", Message: "is invalid"},
				{Field: "", Message: "general error"},
			},
			want: "email: is invalid; general error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errors.Error(); got != tt.want {
				t.Errorf("ValidationErrors.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationErrorsHasErrors(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		want   bool
	}{
		{
			name:   "no errors",
			errors: ValidationErrors{},
			want:   false,
		},
		{
			name: "single error",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
			},
			want: true,
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
				{Field: "password", Message: "is too short"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errors.HasErrors(); got != tt.want {
				t.Errorf("ValidationErrors.HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationErrorsAdd(t *testing.T) {
	tests := []struct {
		name      string
		initial   ValidationErrors
		field     string
		message   string
		wantCount int
	}{
		{
			name:      "add to empty",
			initial:   ValidationErrors{},
			field:     "email",
			message:   "is required",
			wantCount: 1,
		},
		{
			name: "add to existing",
			initial: ValidationErrors{
				{Field: "name", Message: "is required"},
			},
			field:     "email",
			message:   "is invalid",
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.initial
			errors.Add(tt.field, tt.message)

			if len(errors) != tt.wantCount {
				t.Errorf("ValidationErrors.Add() count = %d, want %d", len(errors), tt.wantCount)
			}

			lastError := errors[len(errors)-1]
			if lastError.Field != tt.field || lastError.Message != tt.message {
				t.Errorf("ValidationErrors.Add() last error = %v, want field=%s message=%s",
					lastError, tt.field, tt.message)
			}
		})
	}
}

func TestValidationErrorsAddError(t *testing.T) {
	tests := []struct {
		name      string
		initial   ValidationErrors
		err       ValidationError
		wantCount int
	}{
		{
			name:      "add error to empty",
			initial:   ValidationErrors{},
			err:       ValidationError{Field: "email", Message: "is required"},
			wantCount: 1,
		},
		{
			name: "add error to existing",
			initial: ValidationErrors{
				{Field: "name", Message: "is required"},
			},
			err:       ValidationError{Field: "email", Message: "is invalid"},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.initial
			errors.AddError(tt.err)

			if len(errors) != tt.wantCount {
				t.Errorf("ValidationErrors.AddError() count = %d, want %d", len(errors), tt.wantCount)
			}

			lastError := errors[len(errors)-1]
			if lastError != tt.err {
				t.Errorf("ValidationErrors.AddError() last error = %v, want %v", lastError, tt.err)
			}
		})
	}
}

func TestValidationErrorsMerge(t *testing.T) {
	tests := []struct {
		name      string
		initial   ValidationErrors
		other     ValidationErrors
		wantCount int
	}{
		{
			name:    "merge into empty",
			initial: ValidationErrors{},
			other: ValidationErrors{
				{Field: "email", Message: "is required"},
			},
			wantCount: 1,
		},
		{
			name: "merge empty into existing",
			initial: ValidationErrors{
				{Field: "name", Message: "is required"},
			},
			other:     ValidationErrors{},
			wantCount: 1,
		},
		{
			name: "merge two non-empty",
			initial: ValidationErrors{
				{Field: "name", Message: "is required"},
			},
			other: ValidationErrors{
				{Field: "email", Message: "is required"},
				{Field: "password", Message: "is too short"},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.initial
			errors.Merge(tt.other)

			if len(errors) != tt.wantCount {
				t.Errorf("ValidationErrors.Merge() count = %d, want %d", len(errors), tt.wantCount)
			}
		})
	}
}

func TestValidationErrorsForField(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		field  string
		want   []string
	}{
		{
			name:   "no errors",
			errors: ValidationErrors{},
			field:  "email",
			want:   nil,
		},
		{
			name: "no errors for field",
			errors: ValidationErrors{
				{Field: "name", Message: "is required"},
			},
			field: "email",
			want:  nil,
		},
		{
			name: "single error for field",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
			},
			field: "email",
			want:  []string{"is required"},
		},
		{
			name: "multiple errors for field",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
				{Field: "name", Message: "is too short"},
				{Field: "email", Message: "is invalid"},
			},
			field: "email",
			want:  []string{"is required", "is invalid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errors.ForField(tt.field)

			if len(got) != len(tt.want) {
				t.Errorf("ValidationErrors.ForField() len = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ValidationErrors.ForField()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidationErrorsFields(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		want   []string
	}{
		{
			name:   "no errors",
			errors: ValidationErrors{},
			want:   nil,
		},
		{
			name: "single field",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
			},
			want: []string{"email"},
		},
		{
			name: "multiple fields",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
				{Field: "name", Message: "is required"},
			},
			want: []string{"email", "name"},
		},
		{
			name: "duplicate fields",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
				{Field: "email", Message: "is invalid"},
			},
			want: []string{"email"},
		},
		{
			name: "with empty field",
			errors: ValidationErrors{
				{Field: "email", Message: "is required"},
				{Field: "", Message: "general error"},
				{Field: "name", Message: "is required"},
			},
			want: []string{"email", "name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errors.Fields()

			if len(got) != len(tt.want) {
				t.Errorf("ValidationErrors.Fields() len = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ValidationErrors.Fields()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidatorFuncValidate(t *testing.T) {
	tests := []struct {
		name      string
		validator ValidatorFunc
		wantCount int
	}{
		{
			name: "returns no errors",
			validator: func() ValidationErrors {
				return ValidationErrors{}
			},
			wantCount: 0,
		},
		{
			name: "returns one error",
			validator: func() ValidationErrors {
				return ValidationErrors{
					{Field: "email", Message: "is required"},
				}
			},
			wantCount: 1,
		},
		{
			name: "returns multiple errors",
			validator: func() ValidationErrors {
				var errors ValidationErrors
				errors.Add("email", "is required")
				errors.Add("password", "is too short")
				return errors
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.validator.Validate()

			if len(got) != tt.wantCount {
				t.Errorf("ValidatorFunc.Validate() count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestCombine(t *testing.T) {
	tests := []struct {
		name       string
		validators []Validator
		wantCount  int
	}{
		{
			name:       "no validators",
			validators: []Validator{},
			wantCount:  0,
		},
		{
			name: "single validator with no errors",
			validators: []Validator{
				ValidatorFunc(func() ValidationErrors {
					return ValidationErrors{}
				}),
			},
			wantCount: 0,
		},
		{
			name: "single validator with errors",
			validators: []Validator{
				ValidatorFunc(func() ValidationErrors {
					return ValidationErrors{
						{Field: "email", Message: "is required"},
					}
				}),
			},
			wantCount: 1,
		},
		{
			name: "multiple validators with errors",
			validators: []Validator{
				ValidatorFunc(func() ValidationErrors {
					return ValidationErrors{
						{Field: "email", Message: "is required"},
					}
				}),
				ValidatorFunc(func() ValidationErrors {
					return ValidationErrors{
						{Field: "password", Message: "is too short"},
					}
				}),
			},
			wantCount: 2,
		},
		{
			name: "mixed validators with and without errors",
			validators: []Validator{
				ValidatorFunc(func() ValidationErrors {
					return ValidationErrors{
						{Field: "email", Message: "is required"},
					}
				}),
				ValidatorFunc(func() ValidationErrors {
					return ValidationErrors{}
				}),
				ValidatorFunc(func() ValidationErrors {
					return ValidationErrors{
						{Field: "password", Message: "is too short"},
					}
				}),
			},
			wantCount: 2,
		},
		{
			name: "with nil validator",
			validators: []Validator{
				ValidatorFunc(func() ValidationErrors {
					return ValidationErrors{
						{Field: "email", Message: "is required"},
					}
				}),
				nil,
				ValidatorFunc(func() ValidationErrors {
					return ValidationErrors{
						{Field: "password", Message: "is too short"},
					}
				}),
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Combine(tt.validators...)

			if len(got) != tt.wantCount {
				t.Errorf("Combine() count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestIsRequired(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "valid string",
			value: "hello",
			want:  true,
		},
		{
			name:  "empty string",
			value: "",
			want:  false,
		},
		{
			name:  "only spaces",
			value: "   ",
			want:  false,
		},
		{
			name:  "string with spaces",
			value: "  hello  ",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRequired(tt.value); got != tt.want {
				t.Errorf("IsRequired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRequiredUUID(t *testing.T) {
	tests := []struct {
		name  string
		value uuid.UUID
		want  bool
	}{
		{
			name:  "valid uuid",
			value: uuid.New(),
			want:  true,
		},
		{
			name:  "nil uuid",
			value: uuid.Nil,
			want:  false,
		},
		{
			name:  "zero uuid",
			value: uuid.UUID{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRequiredUUID(tt.value); got != tt.want {
				t.Errorf("IsRequiredUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name  string
		value string
		min   int
		want  bool
	}{
		{
			name:  "exactly minimum",
			value: "hello",
			min:   5,
			want:  true,
		},
		{
			name:  "above minimum",
			value: "hello world",
			min:   5,
			want:  true,
		},
		{
			name:  "below minimum",
			value: "hi",
			min:   5,
			want:  false,
		},
		{
			name:  "empty string",
			value: "",
			min:   1,
			want:  false,
		},
		{
			name:  "zero minimum",
			value: "",
			min:   0,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MinLength(tt.value, tt.min); got != tt.want {
				t.Errorf("MinLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name  string
		value string
		max   int
		want  bool
	}{
		{
			name:  "exactly maximum",
			value: "hello",
			max:   5,
			want:  true,
		},
		{
			name:  "below maximum",
			value: "hi",
			max:   5,
			want:  true,
		},
		{
			name:  "above maximum",
			value: "hello world",
			max:   5,
			want:  false,
		},
		{
			name:  "empty string",
			value: "",
			max:   5,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxLength(tt.value, tt.max); got != tt.want {
				t.Errorf("MaxLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinValueInt(t *testing.T) {
	tests := []struct {
		name  string
		value int
		min   int
		want  bool
	}{
		{
			name:  "exactly minimum",
			value: 5,
			min:   5,
			want:  true,
		},
		{
			name:  "above minimum",
			value: 10,
			min:   5,
			want:  true,
		},
		{
			name:  "below minimum",
			value: 3,
			min:   5,
			want:  false,
		},
		{
			name:  "negative values",
			value: -5,
			min:   -10,
			want:  true,
		},
		{
			name:  "zero minimum",
			value: 0,
			min:   0,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MinValueInt(tt.value, tt.min); got != tt.want {
				t.Errorf("MinValueInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxValueInt(t *testing.T) {
	tests := []struct {
		name  string
		value int
		max   int
		want  bool
	}{
		{
			name:  "exactly maximum",
			value: 5,
			max:   5,
			want:  true,
		},
		{
			name:  "below maximum",
			value: 3,
			max:   5,
			want:  true,
		},
		{
			name:  "above maximum",
			value: 10,
			max:   5,
			want:  false,
		},
		{
			name:  "negative values below max",
			value: -10,
			max:   -5,
			want:  true,
		},
		{
			name:  "negative values above max",
			value: -3,
			max:   -5,
			want:  false,
		},
		{
			name:  "zero maximum",
			value: 0,
			max:   0,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxValueInt(tt.value, tt.max); got != tt.want {
				t.Errorf("MaxValueInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInRange(t *testing.T) {
	tests := []struct {
		name  string
		value int
		min   int
		max   int
		want  bool
	}{
		{
			name:  "within range",
			value: 5,
			min:   1,
			max:   10,
			want:  true,
		},
		{
			name:  "at minimum",
			value: 1,
			min:   1,
			max:   10,
			want:  true,
		},
		{
			name:  "at maximum",
			value: 10,
			min:   1,
			max:   10,
			want:  true,
		},
		{
			name:  "below range",
			value: 0,
			min:   1,
			max:   10,
			want:  false,
		},
		{
			name:  "above range",
			value: 11,
			min:   1,
			max:   10,
			want:  false,
		},
		{
			name:  "negative range",
			value: -5,
			min:   -10,
			max:   -1,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InRange(tt.value, tt.min, tt.max); got != tt.want {
				t.Errorf("InRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOneOf(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		allowed []string
		want    bool
	}{
		{
			name:    "value in list",
			value:   "apple",
			allowed: []string{"apple", "banana", "orange"},
			want:    true,
		},
		{
			name:    "value not in list",
			value:   "grape",
			allowed: []string{"apple", "banana", "orange"},
			want:    false,
		},
		{
			name:    "empty list",
			value:   "apple",
			allowed: []string{},
			want:    false,
		},
		{
			name:    "empty value in list",
			value:   "",
			allowed: []string{"", "apple"},
			want:    true,
		},
		{
			name:    "empty value not in list",
			value:   "",
			allowed: []string{"apple", "banana"},
			want:    false,
		},
		{
			name:    "case sensitive",
			value:   "Apple",
			allowed: []string{"apple", "banana"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OneOf(tt.value, tt.allowed); got != tt.want {
				t.Errorf("OneOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequiredString(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "valid string",
			field:     "email",
			value:     "test@example.com",
			wantError: false,
		},
		{
			name:      "empty string",
			field:     "email",
			value:     "",
			wantError: true,
		},
		{
			name:      "only spaces",
			field:     "email",
			value:     "   ",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequiredString(tt.field, tt.value)

			if (err.Field != "") != tt.wantError {
				t.Errorf("RequiredString() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err.Field != tt.field {
				t.Errorf("RequiredString() field = %v, want %v", err.Field, tt.field)
			}
		})
	}
}

func TestRequiredUUID(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     uuid.UUID
		wantError bool
	}{
		{
			name:      "valid uuid",
			field:     "id",
			value:     uuid.New(),
			wantError: false,
		},
		{
			name:      "nil uuid",
			field:     "id",
			value:     uuid.Nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequiredUUID(tt.field, tt.value)

			if (err.Field != "") != tt.wantError {
				t.Errorf("RequiredUUID() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err.Field != tt.field {
				t.Errorf("RequiredUUID() field = %v, want %v", err.Field, tt.field)
			}
		})
	}
}

func TestStringMinLength(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		min       int
		wantError bool
	}{
		{
			name:      "valid length",
			field:     "password",
			value:     "password123",
			min:       8,
			wantError: false,
		},
		{
			name:      "too short",
			field:     "password",
			value:     "pass",
			min:       8,
			wantError: true,
		},
		{
			name:      "exactly minimum",
			field:     "password",
			value:     "password",
			min:       8,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StringMinLength(tt.field, tt.value, tt.min)

			if (err.Field != "") != tt.wantError {
				t.Errorf("StringMinLength() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err.Field != tt.field {
				t.Errorf("StringMinLength() field = %v, want %v", err.Field, tt.field)
			}
		})
	}
}

func TestStringMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		max       int
		wantError bool
	}{
		{
			name:      "valid length",
			field:     "username",
			value:     "john",
			max:       32,
			wantError: false,
		},
		{
			name:      "too long",
			field:     "username",
			value:     strings.Repeat("a", 40),
			max:       32,
			wantError: true,
		},
		{
			name:      "exactly maximum",
			field:     "username",
			value:     strings.Repeat("a", 32),
			max:       32,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StringMaxLength(tt.field, tt.value, tt.max)

			if (err.Field != "") != tt.wantError {
				t.Errorf("StringMaxLength() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err.Field != tt.field {
				t.Errorf("StringMaxLength() field = %v, want %v", err.Field, tt.field)
			}
		})
	}
}

func TestIntMinValue(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     int
		min       int
		wantError bool
	}{
		{
			name:      "valid value",
			field:     "age",
			value:     25,
			min:       18,
			wantError: false,
		},
		{
			name:      "too low",
			field:     "age",
			value:     15,
			min:       18,
			wantError: true,
		},
		{
			name:      "exactly minimum",
			field:     "age",
			value:     18,
			min:       18,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IntMinValue(tt.field, tt.value, tt.min)

			if (err.Field != "") != tt.wantError {
				t.Errorf("IntMinValue() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err.Field != tt.field {
				t.Errorf("IntMinValue() field = %v, want %v", err.Field, tt.field)
			}
		})
	}
}

func TestIntMaxValue(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     int
		max       int
		wantError bool
	}{
		{
			name:      "valid value",
			field:     "quantity",
			value:     5,
			max:       10,
			wantError: false,
		},
		{
			name:      "too high",
			field:     "quantity",
			value:     15,
			max:       10,
			wantError: true,
		},
		{
			name:      "exactly maximum",
			field:     "quantity",
			value:     10,
			max:       10,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IntMaxValue(tt.field, tt.value, tt.max)

			if (err.Field != "") != tt.wantError {
				t.Errorf("IntMaxValue() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err.Field != tt.field {
				t.Errorf("IntMaxValue() field = %v, want %v", err.Field, tt.field)
			}
		})
	}
}

func TestIntInRange(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     int
		min       int
		max       int
		wantError bool
	}{
		{
			name:      "within range",
			field:     "rating",
			value:     5,
			min:       1,
			max:       10,
			wantError: false,
		},
		{
			name:      "below range",
			field:     "rating",
			value:     0,
			min:       1,
			max:       10,
			wantError: true,
		},
		{
			name:      "above range",
			field:     "rating",
			value:     11,
			min:       1,
			max:       10,
			wantError: true,
		},
		{
			name:      "at minimum",
			field:     "rating",
			value:     1,
			min:       1,
			max:       10,
			wantError: false,
		},
		{
			name:      "at maximum",
			field:     "rating",
			value:     10,
			min:       1,
			max:       10,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IntInRange(tt.field, tt.value, tt.min, tt.max)

			if (err.Field != "") != tt.wantError {
				t.Errorf("IntInRange() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err.Field != tt.field {
				t.Errorf("IntInRange() field = %v, want %v", err.Field, tt.field)
			}
		})
	}
}

func TestStringOneOf(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		allowed   []string
		wantError bool
	}{
		{
			name:      "valid value",
			field:     "status",
			value:     "active",
			allowed:   []string{"active", "inactive", "pending"},
			wantError: false,
		},
		{
			name:      "invalid value",
			field:     "status",
			value:     "deleted",
			allowed:   []string{"active", "inactive", "pending"},
			wantError: true,
		},
		{
			name:      "empty value not in list",
			field:     "status",
			value:     "",
			allowed:   []string{"active", "inactive"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StringOneOf(tt.field, tt.value, tt.allowed)

			if (err.Field != "") != tt.wantError {
				t.Errorf("StringOneOf() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err.Field != tt.field {
				t.Errorf("StringOneOf() field = %v, want %v", err.Field, tt.field)
			}
		})
	}
}
