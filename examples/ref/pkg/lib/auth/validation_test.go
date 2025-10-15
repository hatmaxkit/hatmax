package auth

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		expectedCount int
		expectedCodes []string
	}{
		{
			name:          "valid email",
			email:         "user@example.com",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid email with subdomain",
			email:         "user@mail.example.com",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid email with plus addressing",
			email:         "user+tag@example.com",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid email with numbers",
			email:         "user123@example.com",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "empty email",
			email:         "",
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name:          "email with spaces gets normalized",
			email:         "  user@example.com  ",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "only spaces becomes empty",
			email:         "   ",
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name:          "missing @ symbol",
			email:         "userexample.com",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "missing domain",
			email:         "user@",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "missing local part",
			email:         "@example.com",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "invalid domain format",
			email:         "user@example",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "too long email",
			email:         "verylongusernamethatexceedslimits@verylongdomainnamethatexceedslimitsandcontinuesforever.verylongdomainextensionthatexceedslimitsandcontinuesforeverandeverandeverandeverandeverandeverandeverandeverandeverandeverandeverandeverandeverandeverandeverandeverandeverandever.com",
			expectedCount: 1,
			expectedCodes: []string{"too_long"},
		},
		{
			name:          "too long local part",
			email:         "verylongusernamethatexceedssixtyfourcharactersandcontinuesforever@example.com",
			expectedCount: 1,
			expectedCodes: []string{"local_too_long"},
		},
		{
			name:          "multiple @ symbols",
			email:         "user@@example.com",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "invalid characters",
			email:         "user name@example.com",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateEmail(tt.email)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateEmail() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		expectedCount int
		expectedCodes []string
	}{
		{
			name:          "valid password",
			password:      "Password123!",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid complex password",
			password:      "MySecure@Pass123",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "empty password",
			password:      "",
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name:          "too short password",
			password:      "Pass1!",
			expectedCount: 1,
			expectedCodes: []string{"too_short"},
		},
		{
			name:          "minimum length valid password",
			password:      "Pass123!",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "too long password",
			password:      "VeryLongPasswordThatExceedsOneHundredTwentyEightCharactersAndContinuesForeverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEverAndEver!123A",
			expectedCount: 1,
			expectedCodes: []string{"too_long"},
		},
		{
			name:          "missing uppercase",
			password:      "password123!",
			expectedCount: 1,
			expectedCodes: []string{"missing_uppercase"},
		},
		{
			name:          "missing lowercase",
			password:      "PASSWORD123!",
			expectedCount: 1,
			expectedCodes: []string{"missing_lowercase"},
		},
		{
			name:          "missing digit",
			password:      "Password!",
			expectedCount: 1,
			expectedCodes: []string{"missing_digit"},
		},
		{
			name:          "missing special character",
			password:      "Password123",
			expectedCount: 1,
			expectedCodes: []string{"missing_special"},
		},
		{
			name:          "multiple validation errors",
			password:      "pass",
			expectedCount: 4,
			expectedCodes: []string{"too_short", "missing_uppercase", "missing_digit", "missing_special"},
		},
		{
			name:          "unicode characters valid",
			password:      "Пароль123!",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "various special characters",
			password:      "Password123@#$%",
			expectedCount: 0,
			expectedCodes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidatePassword(tt.password)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidatePassword() returned %d errors, want %d", len(errors), tt.expectedCount)
				for i, err := range errors {
					t.Logf("Error %d: %s - %s", i, err.Code, err.Message)
				}
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestValidateUserStatus(t *testing.T) {
	tests := []struct {
		name          string
		status        UserStatus
		expectedCount int
		expectedCodes []string
	}{
		{
			name:          "valid active status",
			status:        UserStatusActive,
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid suspended status",
			status:        UserStatusSuspended,
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid deleted status",
			status:        UserStatusDeleted,
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "invalid status",
			status:        UserStatus("invalid"),
			expectedCount: 1,
			expectedCodes: []string{"invalid_value"},
		},
		{
			name:          "empty status",
			status:        UserStatus(""),
			expectedCount: 1,
			expectedCodes: []string{"invalid_value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateUserStatus(tt.status)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateUserStatus() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestValidateGrantType(t *testing.T) {
	tests := []struct {
		name          string
		grantType     GrantType
		expectedCount int
		expectedCodes []string
	}{
		{
			name:          "valid role grant type",
			grantType:     GrantTypeRole,
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid permission grant type",
			grantType:     GrantTypePermission,
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "invalid grant type",
			grantType:     GrantType("invalid"),
			expectedCount: 1,
			expectedCodes: []string{"invalid_value"},
		},
		{
			name:          "empty grant type",
			grantType:     GrantType(""),
			expectedCount: 1,
			expectedCodes: []string{"invalid_value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateGrantType(tt.grantType)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateGrantType() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestValidateScope(t *testing.T) {
	tests := []struct {
		name          string
		scope         Scope
		expectedCount int
		expectedCodes []string
	}{
		{
			name:          "valid global scope",
			scope:         Scope{Type: "global", ID: ""},
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid team scope",
			scope:         Scope{Type: "team", ID: "123"},
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid organization scope",
			scope:         Scope{Type: "organization", ID: "456"},
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "empty scope type",
			scope:         Scope{Type: "", ID: "123"},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name:          "invalid scope type",
			scope:         Scope{Type: "invalid", ID: "123"},
			expectedCount: 1,
			expectedCodes: []string{"invalid_value"},
		},
		{
			name:          "global scope with ID",
			scope:         Scope{Type: "global", ID: "should-be-empty"},
			expectedCount: 1,
			expectedCodes: []string{"invalid_for_global"},
		},
		{
			name:          "team scope without ID",
			scope:         Scope{Type: "team", ID: ""},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name:          "organization scope without ID",
			scope:         Scope{Type: "organization", ID: ""},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name:          "multiple errors",
			scope:         Scope{Type: "", ID: ""},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateScope(tt.scope)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateScope() returned %d errors, want %d", len(errors), tt.expectedCount)
				for i, err := range errors {
					t.Logf("Error %d: %s - %s", i, err.Code, err.Message)
				}
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestValidatePermissionCode(t *testing.T) {
	tests := []struct {
		name          string
		permission    string
		expectedCount int
		expectedCodes []string
	}{
		{
			name:          "valid permission code",
			permission:    "orders:read",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid permission with underscores",
			permission:    "user_profiles:manage",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "valid permission with numbers",
			permission:    "orders2:read3",
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name:          "empty permission",
			permission:    "",
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name:          "too long permission",
			permission:    "very_long_resource_name_that_exceeds_one_hundred_characters_and_continues_forever_and_ever_and_ever:action_name_that_also_continues",
			expectedCount: 1,
			expectedCodes: []string{"too_long"},
		},
		{
			name:          "missing colon",
			permission:    "ordersread",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "multiple colons",
			permission:    "orders:read:extra",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "missing resource part",
			permission:    ":read",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "missing action part",
			permission:    "orders:",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "uppercase characters",
			permission:    "Orders:Read",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "invalid characters",
			permission:    "orders-read:action",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "spaces in permission",
			permission:    "orders read:action",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "resource starts with number",
			permission:    "2orders:read",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
		{
			name:          "action starts with number",
			permission:    "orders:2read",
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidatePermissionCode(tt.permission)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidatePermissionCode() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}
