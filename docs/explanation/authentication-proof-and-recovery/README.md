<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication Proof and Recovery

An account, an enrolled authenticator and a session answer different questions.
The account identifies the subject. Enrollment records a verifier. A completed
authentication records what that subject actually proved at a particular time.
None of those facts alone grants permission to modify an application resource.

## Enrollment Does Not Establish Access

An enrollment token authorizes one restricted ceremony. It is neither a session
nor evidence that the newly created factor has completed sign-in. Separating
registration from assertion keeps a setup response from becoming stronger access
without an actual proof operation.

Established-factor changes need current authority from an existing session.
They cannot reuse the weaker initial-setup path. Replacement commits creation
and removal together, so an unsuccessful new ceremony does not destroy the old
factor. The last usable primary-factor rule keeps management from silently
removing the ability to satisfy the current profile.

## Proof Has a Method, Age and Binding

Password proof cannot satisfy an MFA requirement. Password plus TOTP or a backup
code supplies MFA, but lacks WebAuthn's relying-party binding. The supported
user-verified WebAuthn profile supplies phishing-resistant MFA. A route chooses
the required method and policy revision in trusted application code.

Proof also ages. Relevant activity can refresh inactivity within its configured
cadence; it cannot refresh authentication time or extend absolute expiry. A
recent-proof operation therefore needs a real reauthentication or step-up even
when the session remains usable for ordinary work.

Factor-backed proof remains bound to current factor state. Removing or replacing
its constituent invalidates that authority. Security changes retire stale
sessions and pending ceremonies atomically. Rotating an allowed retained actor
changes its bearer without pretending its proof is younger.

## Recovery Preserves Stronger Requirements

Mailbox verification establishes control of the current address. A password
reset replaces the password after a separate one-use mailed challenge. Neither
operation proves possession of an enrolled authenticator, grants a new session
or removes an existing MFA requirement.

This is why password reset cannot serve as recovery from losing every required
factor. An application needs a separately designed identity-recovery policy for
that problem. Backup codes are prepared fallback proof, not permission to
rewrite the account's stronger management policy.

## The Database Owns the Final Decision

A handler's earlier check can become stale while a request waits for locks or
verifies cryptographic proof. The typed storage contracts recheck current
subject version, actor generation, factor bindings, policy and trusted time
after waiting and before committing. Shared admission is spent independently
of later mutation success, so failure cannot refund guesses or reset a budget.

A committed password change remains committed if notification delivery fails.
Likewise, a lost HTTP response does not establish rollback. The application
must observe current state before retrying an uncertain mutation. Best-effort
security observations support diagnosis; they are not a durable audit outbox.

Hatmax owns these proof and storage contracts. The application owns its adapter,
trusted route requirements, origin and keys, mail delivery and resource-specific
authorization. The [authentication reference](../../reference/authentication/README.md)
states those contracts; [Manage Authenticators](../../how-to/manage-authenticators/README.md)
shows the corresponding task procedures.
