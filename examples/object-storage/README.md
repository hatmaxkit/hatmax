<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Private Object Storage Examples

The [local tutorial](../../docs/tutorials/private-objects/README.md) runs
`./examples/object-storage/local` against one existing owned directory.
The [signed-upload procedure](../../docs/how-to/sign-object-uploads/README.md)
runs `./examples/object-storage/remote` against an existing private, unversioned
MinIO bucket. Each command reports only a fixed success or failure message.

The remote example uses constructor-owned `OBJECT_STORAGE_BUCKET`,
`OBJECT_STORAGE_REGION`, `OBJECT_STORAGE_ENDPOINT` and `OBJECT_STORAGE_SPOOL`.
Credentials come from the existing SDK chain. `OBJECT_STORAGE_FIXTURE_LOOPBACK=1`
is solely for an owned literal-loopback HTTP fixture. Production uses verified
HTTPS and the named tested POST profile. The demo owns and may replace
`guide/staging-document.txt` and `guide/final-document.txt`; use an isolated bucket.
It deletes both keys on exit and closes readers, spool handles and connections.

The examples intentionally allocate at most one 1 MiB validated snapshot.
The small multipart demonstration buffers a fixed document plus bounded signer
fields. General large uploads need caller-owned bounded streaming/spooling.
Applications supply admission, current-authority checks, aggregate quotas,
format validation and durable completion; these commands have no application
identity or persistence layer.

Run `GOWORK=off scripts/check-object-storage.sh slice 4` from the repository
root to execute the published local and actual signed-remote workflows. The
runner owns and identifies its native server, bucket and temporary resources;
no external account or existing server is used.
