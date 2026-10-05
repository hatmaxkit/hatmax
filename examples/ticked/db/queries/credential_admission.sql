-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- name: EnsureCredentialAdmissionCapacity :exec
INSERT INTO credential_admission_capacity(namespace,key_binding,records,max_records)
VALUES ($1,$2,0,$3) ON CONFLICT DO NOTHING;

-- name: LockCredentialAdmissionCapacity :one
SELECT * FROM credential_admission_capacity WHERE namespace=$1 FOR UPDATE;

-- name: UpdateCredentialAdmissionCapacity :execrows
UPDATE credential_admission_capacity SET records=$2,max_records=$3 WHERE namespace=$1;

-- name: LockCredentialAdmission :one
SELECT * FROM credential_admissions WHERE namespace=$1 AND purpose=$2 AND identity_key=$3 FOR UPDATE;

-- name: InsertCredentialAdmission :exec
INSERT INTO credential_admissions(namespace,purpose,identity_key,window_start,window_end,cooldown_us,attempt_limit,attempts,cooldown_until)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9);

-- name: UpdateCredentialAdmission :execrows
UPDATE credential_admissions SET window_start=$4,window_end=$5,cooldown_us=$6,attempt_limit=$7,attempts=$8,cooldown_until=$9
WHERE namespace=$1 AND purpose=$2 AND identity_key=$3;

-- name: DeleteRetiredCredentialAdmissions :execrows
WITH retired AS (
 SELECT a.namespace,a.purpose,a.identity_key FROM credential_admissions a
 WHERE a.namespace=sqlc.arg(namespace) AND GREATEST(a.window_end,COALESCE(a.cooldown_until,a.window_end))<=sqlc.arg(now)::timestamptz
 ORDER BY GREATEST(a.window_end,COALESCE(a.cooldown_until,a.window_end)),a.purpose,a.identity_key
 LIMIT sqlc.arg(cleanup_batch)::integer FOR UPDATE SKIP LOCKED
)
DELETE FROM credential_admissions USING retired WHERE credential_admissions.namespace=retired.namespace
AND credential_admissions.purpose=retired.purpose AND credential_admissions.identity_key=retired.identity_key;
