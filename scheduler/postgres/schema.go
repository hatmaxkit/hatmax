// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

const Schema = `
CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    task_type TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    payload JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    schedule_tz TEXT NOT NULL DEFAULT 'UTC',
    schedule_spec TEXT NOT NULL DEFAULT '',
    next_run_at TIMESTAMPTZ NOT NULL,
    last_run_at TIMESTAMPTZ,
    misfire_policy TEXT DEFAULT 'run',
    max_retries INT DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_due ON scheduled_jobs(enabled, next_run_at);

CREATE TABLE IF NOT EXISTS job_runs (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES scheduled_jobs(id) ON DELETE CASCADE,
    scheduled_for TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending',
    attempt INT NOT NULL DEFAULT 1,
    retry_at TIMESTAMPTZ,
    retry_limit INT,
    error TEXT,
    output JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(job_id, scheduled_for)
);

CREATE INDEX IF NOT EXISTS idx_job_runs_status ON job_runs(status);
CREATE INDEX IF NOT EXISTS idx_job_runs_job_id ON job_runs(job_id);

-- Upgrade installations created before retry waits were supported.
ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS retry_at TIMESTAMPTZ;
ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS retry_limit INT;
` + DurableSchema

// DurableSchema is additive; conditional constraints leave legacy rows intact.
const DurableSchema = `
ALTER TABLE scheduled_jobs ADD COLUMN IF NOT EXISTS durable_policy JSONB;
ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS durable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS handler_timeout_ms INT;
ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS lease_duration_ms INT;
ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS retry_delay_ms INT;
ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS lease_token TEXT;
ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ;
ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS failure_code TEXT;
DO $$ BEGIN
 ALTER TABLE job_runs ADD CONSTRAINT job_runs_durable_policy CHECK (NOT durable OR (
  handler_timeout_ms IS NOT NULL AND handler_timeout_ms BETWEEN 1000 AND 600000
  AND lease_duration_ms IS NOT NULL AND lease_duration_ms BETWEEN handler_timeout_ms + 5000 AND 900000
  AND retry_delay_ms IS NOT NULL AND retry_delay_ms BETWEEN 1000 AND 3600000
  AND retry_limit IS NOT NULL AND retry_limit BETWEEN 1 AND 32 AND attempt BETWEEN 1 AND retry_limit
  AND error IS NULL
  AND (failure_code IS NULL OR failure_code ~ '^[a-z_]{1,64}$')
  AND (output IS NULL OR octet_length(output::text) <= 8192)
  AND ((status = 'running' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL AND retry_at IS NULL AND finished_at IS NULL)
   OR (status = 'retry_wait' AND lease_token IS NULL AND lease_expires_at IS NULL AND retry_at IS NOT NULL AND finished_at IS NULL)
   OR (status IN ('success','failed') AND lease_token IS NULL AND lease_expires_at IS NULL AND retry_at IS NULL AND finished_at IS NOT NULL))
 ));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
CREATE INDEX IF NOT EXISTS idx_job_runs_lease ON job_runs(lease_expires_at) WHERE durable AND status = 'running';
CREATE INDEX IF NOT EXISTS idx_job_runs_retry ON job_runs(retry_at) WHERE status = 'retry_wait';
CREATE INDEX IF NOT EXISTS idx_job_runs_terminal ON job_runs(finished_at, id) WHERE status IN ('success','failed');
`
