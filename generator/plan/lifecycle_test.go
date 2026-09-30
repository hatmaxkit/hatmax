// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package plan

import (
	"sync"
	"testing"

	"hatmax.adrianpk.com/generator/project"
)

func TestLifecycleClaimsFreshPlanOnce(t *testing.T) {
	value, context := expandedTestPlan(t)

	lifecycle, err := Activate(value)
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	if lifecycle.State() != LifecycleReady {
		t.Fatalf("State() = %q, want %q", lifecycle.State(), LifecycleReady)
	}

	result, err := lifecycle.Claim(context.Fingerprint)
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}

	if result.State != LifecycleConsumed || len(result.Diagnostics) != 0 {
		t.Errorf("Claim() = %#v, want consumed plan", result)
	}

	second, err := lifecycle.Claim(context.Fingerprint)
	if err != nil {
		t.Fatalf("second Claim() error = %v", err)
	}

	if second.State != LifecycleConsumed || !hasPlanDiagnostic(second.Diagnostics, "HMGEN-PLAN-USED") {
		t.Errorf("second Claim() = %#v, want plan-used diagnostic", second)
	}
}

func TestLifecycleInvalidatesStalePlan(t *testing.T) {
	value, context := expandedTestPlan(t)

	lifecycle, err := Activate(value)
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	current := context.Fingerprint
	current.Value = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	current.Observations = cloneObservations(current.Observations)
	current.Observations[0].Digest = "changed"

	result, err := lifecycle.Claim(current)
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}

	if result.State != LifecycleStale || !hasPlanDiagnostic(result.Diagnostics, "HMGEN-PLAN-STALE") || len(result.Changes) != 1 {
		t.Errorf("Claim() = %#v, want stale plan with one change", result)
	}

	if lifecycle.State() != LifecycleStale {
		t.Errorf("State() = %q, want %q", lifecycle.State(), LifecycleStale)
	}
}

func TestLifecycleRejectsReadyPlan(t *testing.T) {
	value, context := expandedTestPlan(t)

	lifecycle, err := Activate(value)
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	result := lifecycle.Reject()
	if result.State != LifecycleRejected || len(result.Diagnostics) != 0 {
		t.Errorf("Reject() = %#v, want rejected plan", result)
	}

	claimed, err := lifecycle.Claim(context.Fingerprint)
	if err != nil {
		t.Fatalf("Claim() after Reject() error = %v", err)
	}

	if claimed.State != LifecycleRejected || !hasPlanDiagnostic(claimed.Diagnostics, "HMGEN-PLAN-USED") {
		t.Errorf("Claim() after Reject() = %#v, want plan-used diagnostic", claimed)
	}
}

func TestLifecycleDetachesPlanStorage(t *testing.T) {
	value, _ := expandedTestPlan(t)

	lifecycle, err := Activate(value)
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	value.Capabilities[0] = "changed"
	detached := lifecycle.Plan()
	detached.Operations[0].Surfaces[0] = "changed"

	fresh := lifecycle.Plan()
	if fresh.Capabilities[0] == "changed" || fresh.Operations[0].Surfaces[0] == "changed" {
		t.Error("Lifecycle exposed mutable plan storage")
	}

	err = VerifyDigest(fresh)
	if err != nil {
		t.Errorf("VerifyDigest(Lifecycle.Plan()) error = %v", err)
	}
}

func TestLifecycleClaimIsAtomic(t *testing.T) {
	value, context := expandedTestPlan(t)

	lifecycle, err := Activate(value)
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	const claimants = 8

	results := make(chan TransitionResult, claimants)

	var wait sync.WaitGroup
	wait.Add(claimants)

	for range claimants {
		go func() {
			defer wait.Done()

			result, claimErr := lifecycle.Claim(context.Fingerprint)
			if claimErr != nil {
				results <- TransitionResult{Diagnostics: []Diagnostic{{Code: claimErr.Error()}}}

				return
			}

			results <- result
		}()
	}

	wait.Wait()
	close(results)

	consumed := 0
	used := 0

	for result := range results {
		switch {
		case result.State == LifecycleConsumed && len(result.Diagnostics) == 0:
			consumed++
		case hasPlanDiagnostic(result.Diagnostics, "HMGEN-PLAN-USED"):
			used++
		default:
			t.Errorf("unexpected concurrent Claim() result: %#v", result)
		}
	}

	if consumed != 1 || used != claimants-1 {
		t.Errorf("concurrent claims: consumed = %d, used = %d; want 1 and %d", consumed, used, claimants-1)
	}
}

func TestActivateRequiresValidDigest(t *testing.T) {
	_, err := Activate(validTestPlan())
	requirePlanCode(t, err, "plan_digest_missing")

	value, _ := expandedTestPlan(t)
	value.ExpectedObservations = []project.Observation{}

	_, err = Activate(value)
	requirePlanCode(t, err, "plan_digest_mismatch")
}
