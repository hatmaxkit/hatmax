package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DecodeInterpretation strictly decodes one bounded model result. Semantic
// validation against the current project and Book remains Evaluate's job.
func DecodeInterpretation(data []byte) (Interpretation, error) {
	if len(data) > MaximumInterpretationBytes {
		return Interpretation{}, evaluationError("evaluation_output_too_large", "output", "interpretation output exceeds %d bytes", MaximumInterpretationBytes)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return Interpretation{}, evaluationError("evaluation_output_invalid", "output", "interpretation output is required")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var result Interpretation

	err := decoder.Decode(&result)
	if err != nil {
		return Interpretation{}, evaluationError("evaluation_output_invalid", "output", "decode interpretation: %v", err)
	}

	var trailing any

	err = decoder.Decode(&trailing)
	if err != io.EOF {
		if err != nil {
			return Interpretation{}, evaluationError("evaluation_output_invalid", "output", "decode trailing interpretation content: %v", err)
		}

		return Interpretation{}, evaluationError("evaluation_output_invalid", "output", "multiple JSON values are not allowed")
	}

	err = validateInterpretation(result)
	if err != nil {
		return Interpretation{}, fmt.Errorf("decode interpretation: %w", err)
	}

	return cloneInterpretation(result), nil
}
