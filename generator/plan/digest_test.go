package plan

import "testing"

func TestSealProducesAndVerifiesDeterministicDigest(t *testing.T) {
	value := validTestPlan()

	first, err := Seal(value)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	second, err := Seal(value)
	if err != nil {
		t.Fatalf("second Seal() error = %v", err)
	}

	if first.Digest == "" || first.Digest != second.Digest {
		t.Errorf("Seal() digests = %q and %q, want equal non-empty values", first.Digest, second.Digest)
	}

	err = VerifyDigest(first)
	if err != nil {
		t.Errorf("VerifyDigest() error = %v", err)
	}

	resealed, err := Seal(first)
	if err != nil {
		t.Fatalf("Seal(sealed) error = %v", err)
	}

	if resealed.Digest != first.Digest {
		t.Errorf("Seal(sealed).Digest = %q, want %q", resealed.Digest, first.Digest)
	}
}

func TestVerifyDigestRejectsMissingMalformedAndChangedDigests(t *testing.T) {
	value := validTestPlan()
	requirePlanCode(t, VerifyDigest(value), "plan_digest_missing")

	value.Digest = "invalid"
	requirePlanCode(t, VerifyDigest(value), "plan_digest_invalid")

	sealed, err := Seal(validTestPlan())
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	sealed.Feature = "changed"
	requirePlanCode(t, VerifyDigest(sealed), "plan_digest_mismatch")
	requirePlanCode(t, func() error {
		_, marshalErr := MarshalYAML(sealed)

		return marshalErr
	}(), "plan_digest_mismatch")
}

func TestExpandedPlansAreSealed(t *testing.T) {
	admission, context := admittedExpansion(t, "create_feature", []string{"postgres_persistence"})

	result, err := Expand(admission, context)
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}

	if result.Digest == "" {
		t.Fatal("Expand() returned an unsealed plan")
	}

	err = VerifyDigest(result)
	if err != nil {
		t.Errorf("VerifyDigest(expanded plan) error = %v", err)
	}
}
