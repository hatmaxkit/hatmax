package execute

import "testing"

func TestSealProducesAndVerifiesDeterministicDigest(t *testing.T) {
	first, err := Seal(validTestManifest())
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	second, err := Seal(validTestManifest())
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

	first.Edits[0].Target = "internal/feat/invoice/changed.go"
	requireExecutionCode(t, VerifyDigest(first), "execution_digest_mismatch")
}
