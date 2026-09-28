package intent

import (
	"fmt"
	"strings"

	"hatmax.adrianpk.com/generator/book"
)

func validateExceptions(value Intent, selectedBook *book.Book, selection book.Selection) *Result {
	if len(value.Exceptions) == 0 {
		return nil
	}

	selectedRules := make(map[string]struct{}, len(selection.Rules))
	for _, rule := range selection.Rules {
		selectedRules[rule.ID] = struct{}{}
	}

	diagnostics := make([]Diagnostic, 0)

	for index, exception := range value.Exceptions {
		prefix := fmt.Sprintf("exceptions[%d]", index)
		if _, exists := selectedBook.Rule(exception.Rule); !exists {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "HMGEN-EXCEPTION-RULE",
				Field:   prefix + ".rule",
				Message: fmt.Sprintf("exception rule %q does not exist", exception.Rule),
			})
		} else if _, applies := selectedRules[exception.Rule]; !applies {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "HMGEN-EXCEPTION-RULE",
				Field:   prefix + ".rule",
				Message: fmt.Sprintf("exception rule %q does not apply to the selected context", exception.Rule),
			})
		}

		if strings.TrimSpace(exception.Reason) == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "HMGEN-EXCEPTION-REASON",
				Field:   prefix + ".reason",
				Message: "exception reason is required",
			})
		}

		if strings.TrimSpace(exception.Scope) == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "HMGEN-EXCEPTION-SCOPE",
				Field:   prefix + ".scope",
				Message: "exception scope is required",
			})
		}

		if !exception.Approved {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "HMGEN-EXCEPTION-APPROVAL",
				Field:   prefix + ".approved",
				Message: "exception requires explicit user approval",
			})
		}
	}

	if len(diagnostics) == 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "HMGEN-EXCEPTION-EXECUTION-UNSUPPORTED",
			Field:   "exceptions",
			Message: "the first generator delivery parses exceptions but cannot execute them",
		})
	}

	sortDiagnostics(diagnostics)

	return &Result{
		Status:      StatusExceptionRequired,
		Intent:      value,
		Selection:   selection,
		Diagnostics: diagnostics,
	}
}
