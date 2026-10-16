<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Private Object Ownership, Bearers and Snapshots

Object storage separates transport from application authority. A store observes
and moves bytes under constructor-owned configuration. The application decides
who may upload, who owns a key, aggregate quotas, permitted formats, completion
and durable recovery. Private storage does not require an image record or a
public URL. Historical image stores retain their image and URL contracts.

## Metadata describes an observation

`Object` carries a key, byte size and content type. Local content type is derived
from at most 512 bytes on each observation; remote content type is supplied or
provider-reported metadata. Neither certifies a safe file. A remote ETag is not a
content hash. `Stat` does not reserve the bytes that a future Open will observe.
Only successful EOF of one bounded opened reader establishes a complete snapshot;
close ownership stays with the caller.

## A signed form is a temporary bearer

Anyone holding the form can exercise its exact target/type/byte/expiry policy.
It may be reused until expiry and cannot consult current application membership.
Expiry limits provider admission, not application completion or an earlier
in-flight transfer. There is no one-use or immediate-revocation promise.

The URL and fields are sensitive limited grants. Credential identifiers and
required session tokens may appear, but secret access keys and reusable signing
material do not. Examples keep fields in memory only and report fixed outcomes.
Application permission checks happen before issuance and again at completion.

## Completion binds the bytes actually validated

A still-valid bearer may replace staging after a separate metadata inspection.
Reading one opened response to bounded EOF and validating that snapshot avoids
using a later staging version accidentally. Commit those same bytes to a
distinct application-controlled final destination. Choose immutable keys or
coordinate replacement when concurrent writers matter. A store's atomic local
replacement is neither compare-and-swap nor crash-durable application publication.

The application owns the durable completion transition and reconciliation after
uncertain remote writes. No generic object store can turn transport success into
current authorization or a database transaction. Per-operation limits likewise
do not implement process-wide memory, disk or concurrency admission.

Follow the [local tutorial](../../tutorials/private-objects/README.md),
[signed upload procedure](../../how-to/sign-object-uploads/README.md),
[reconciliation procedure](../../how-to/reconcile-object-storage/README.md) or
[exact reference](../../reference/storage/README.md) for the corresponding task.
