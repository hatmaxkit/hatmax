package plan

import (
	"crypto/sha256"
	"encoding/hex"
)

// Seal validates a plan and binds its semantic content to a deterministic
// digest. A previously sealed plan is verified rather than silently resealed.
func Seal(value Plan) (Plan, error) {
	if value.Digest != "" {
		err := VerifyDigest(value)
		if err != nil {
			return Plan{}, err
		}

		return value, nil
	}

	err := Validate(value)
	if err != nil {
		return Plan{}, err
	}

	digest, err := computeDigest(value)
	if err != nil {
		return Plan{}, err
	}

	value.Digest = digest

	return value, nil
}

// VerifyDigest confirms that a sealed plan still matches its semantic
// content.
func VerifyDigest(value Plan) error {
	if value.Digest == "" {
		return planError("plan_digest_missing", "digest", "sealed plan digest is required")
	}

	err := Validate(value)
	if err != nil {
		return err
	}

	want := value.Digest
	value.Digest = ""

	actual, err := computeDigest(value)
	if err != nil {
		return err
	}

	if actual != want {
		return planError("plan_digest_mismatch", "digest", "plan content does not match digest")
	}

	return nil
}

func computeDigest(value Plan) (string, error) {
	data, err := canonicalJSON(value)
	if err != nil {
		return "", err
	}

	digest := sha256.Sum256(data)

	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
