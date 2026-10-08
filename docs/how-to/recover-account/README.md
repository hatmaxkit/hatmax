<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Verify a Mailbox and Recover a Password

Use the Ticked example's account forms to verify the current mailbox or change
a password. Configure active mail delivery and `TICKED_RECOVERY_ORIGIN` with
the application's trusted HTTPS origin. Localhost HTTP requires the explicit
development configuration. These routes remain disabled without active mail.
See [Ticked setup](../../../examples/ticked/README.md) for the runtime settings.

## Verify the Current Mailbox

Open `/account/mailbox`, enter the account email and request verification.
Open the received link with JavaScript enabled, then explicitly submit the
confirmation form. GET only displays it; opening the link alone changes nothing.
The served script removes the secret from the fragment before a same-origin POST.

After successful verification, sign in normally. Previous sessions and unfinished
security ceremonies are retired. Verification does not activate the account,
remove MFA or issue a replacement session. If the link expired or was replaced,
request a new one. The neutral acknowledgment does not guarantee delivery.

## Change a Known Password

Sign in recently and open `/account/password`. Without primary factors, Ticked
allows recent password proof. With factors, confirm a passkey first: password
reauthentication or fallback proof does not meet Ticked's stronger change policy.
Applications choosing another supported MFA policy must wire it in trusted code.

Enter the complete new password and submit. After success, sign in normally
with the new password and required factors. All previous sessions and pending
security work are invalidated. Verified mailbox state, current authenticators
and consumed backup-code state remain intact.

## Reset a Forgotten Password

Open `/account/password/reset` and enter the current previously verified email.
The response does not reveal eligibility. Open the latest received link and
explicitly submit a complete new password. A preview or GET cannot reset it.

Success signs out every session and cancels unfinished security work. Sign in
normally with the new password and any required MFA. An expired, replaced or
consumed link cannot complete again; request a new link when appropriate.

An authorized Ticked operator can initiate the same mailed flow at
`/admin/password-reset` after recent phishing-resistant authentication. The
operator cannot receive the secret, choose the password or create user access.

## Check an Uncertain Completion

If the completion response was lost, try normal sign-in with the intended new
password before requesting another link. A failed notification cannot undo a
committed password change. Do not replay a consumed completion automatically.

Password reset does not replace a lost required authenticator. Use previously
prepared fallback when policy permits; all-factor loss requires a separate
identity-recovery workflow.

Exact contracts are in [Mailbox Verification](../../reference/authentication/README.md#mailbox-verification),
[Password Change](../../reference/authentication/README.md#recent-proof-password-change)
and [Password Reset](../../reference/authentication/README.md#mailbox-password-reset).
For rationale, read
[Authentication Proof and Recovery](../../explanation/authentication-proof-and-recovery/README.md).
