-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- +migrate Up
CREATE TABLE credential_admission_capacity (
 namespace TEXT PRIMARY KEY CHECK (namespace ~ '^[A-Za-z0-9._-]{1,64}$'),
 key_binding BYTEA NOT NULL CHECK (octet_length(key_binding)=32),
 records INTEGER NOT NULL CHECK (records BETWEEN 0 AND 100000),
 max_records INTEGER NOT NULL CHECK (max_records BETWEEN 100 AND 100000)
);
CREATE TABLE credential_admissions (
 namespace TEXT NOT NULL REFERENCES credential_admission_capacity(namespace),
 purpose SMALLINT NOT NULL CHECK (purpose IN (1,2)),
 identity_key BYTEA NOT NULL CHECK (octet_length(identity_key)=32),
 window_start TIMESTAMPTZ NOT NULL,
 window_end TIMESTAMPTZ NOT NULL,
 cooldown_us BIGINT NOT NULL CHECK (cooldown_us BETWEEN 60000000 AND 3600000000),
 attempt_limit INTEGER NOT NULL CHECK (attempt_limit BETWEEN 1 AND 20 AND (purpose=2 OR attempt_limit<=10)),
 attempts INTEGER NOT NULL CHECK (attempts BETWEEN 1 AND attempt_limit),
 cooldown_until TIMESTAMPTZ,
 PRIMARY KEY (namespace,purpose,identity_key),
 CHECK (window_end-window_start BETWEEN INTERVAL '1 minute' AND INTERVAL '1 hour'),
 CHECK (cooldown_until IS NULL OR cooldown_until>window_start)
);
CREATE INDEX credential_admissions_retirement ON credential_admissions
 (namespace, (GREATEST(window_end,COALESCE(cooldown_until,window_end))),purpose,identity_key);

-- +migrate Down
DROP TABLE credential_admissions;
DROP TABLE credential_admission_capacity;
