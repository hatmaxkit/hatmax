<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Manage Authenticators

Use this procedure to manage an established account in the Ticked example.
Start with a running [Ticked application](../../../examples/ticked/README.md),
its explicit WebAuthn RP/origin settings and an enrolled passkey. Use the same
browser origin throughout. App and backup actions also require the configured
`TICKED_TOTP_KEY_ID` and `TICKED_TOTP_KEY`; absent keys disable those actions.

## Confirm the Acting Session

Sign in with the passkey or complete passkey step-up, then open
`/authenticators/manage`. Ticked requires recent phishing-resistant proof.
An ordinary password session, an enrollment bearer or a password-plus-backup
session cannot authorize these changes.

If proof is stale, repeat the actual assertion. Reloading a page does not
refresh proof. Keep the displayed factor list current before selecting a key.

## Add or Replace a Passkey

Choose addition or select the exact factor to replace. Complete the browser's
registration prompt through the served page. Replacement commits the new
credential and removes the old one together; an unsuccessful ceremony leaves
the existing credential available.

After success, reload the factor list. Other sessions and unfinished ceremonies
are retired. Replacing the factor that proved the acting session signs that
session out too; sign in again with the new credential. A retained actor receives
a replacement cookie without gaining newer proof or later expiry.

## Add or Replace an Authenticator App

Begin app setup from the management page. Import the private provisioning URI
into the authenticator app and submit its actual six-digit code. The old app
remains usable until replacement commits. Displaying a setup secret alone does
not enable the new app.

Wait for the next time step before using the app for another operation: the
confirmation consumes its accepted step. Keep the provisioning secret out of
logs, messages and shared screenshots.

## Remove a Factor

Select the current owned factor and remove it. At least one usable primary
factor must remain. The phishing-resistant profile requires a usable WebAuthn
factor; backup codes do not meet that requirement. If removal would eliminate
the last suitable factor, add and confirm its replacement first.

Removing the factor used by the acting session signs it out. Verify the result
through the current list after normal sign-in. A factor ID or security revision
is selection metadata and never grants authority by itself.

## Regenerate Backup Codes

Use backup issuance with recent passkey proof. Store the returned codes
privately when they appear. Issuance replaces the previous set, rotates the
acting cookie and signs out other sessions. Reload cannot redisplay the codes.

For fallback sign-in, submit the password and one exact unused code through the
served flow. Preserve its case and encoding. A successful access completion
consumes that code once; it supplies MFA without phishing resistance and cannot
authorize regeneration of its own backup set.

## Check the Result After an Interrupted Response

Do not automatically resubmit a mutation whose response was lost. Sign in with
the intended surviving factor and inspect the current list. For lost backup
display, confirm a passkey again and explicitly generate a new set. If no
required factor remains accessible, password reset cannot replace it.

For caller integration, use the
[factor contract](../../reference/authentication/README.md#established-authenticator-changes)
and [fallback contract](../../reference/authentication/README.md#totp-and-backup-proof).
For the reasoning, read
[Authentication Proof and Recovery](../../explanation/authentication-proof-and-recovery/README.md).
