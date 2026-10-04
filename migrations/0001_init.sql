-- cf-daily schema.
--
-- This file documents the schema the application expects and adds the
-- constraints the code relies on. It is written to be safe to run against the
-- existing database: every statement is idempotent and nothing is dropped or
-- rewritten.
--
-- Review the unique indexes before applying. If any of them fails, the table
-- already holds duplicate rows that need to be cleaned up first - which is
-- exactly the state these indexes exist to prevent.

BEGIN;

CREATE TABLE IF NOT EXISTS telegram_users (
    id                     BIGSERIAL PRIMARY KEY,
    telegram_user_id       BIGINT      NOT NULL UNIQUE,
    chat_id                BIGINT      NOT NULL,
    username               TEXT        NOT NULL DEFAULT '',
    first_name             TEXT        NOT NULL DEFAULT '',
    active                 BOOLEAN     NOT NULL DEFAULT TRUE,
    github_user_id         BIGINT,
    github_username        TEXT,
    github_installation_id BIGINT,
    github_connected_at    TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS daily_problems (
    id            BIGSERIAL PRIMARY KEY,
    assigned_date DATE    NOT NULL,
    contest_id    INTEGER NOT NULL,
    problem_index TEXT    NOT NULL,
    name          TEXT    NOT NULL,
    rating        INTEGER NOT NULL,
    url           TEXT    NOT NULL,
    tags          TEXT[]  NOT NULL DEFAULT '{}'
);

-- Required by the ON CONFLICT (assigned_date) clause in DailyProblemRepository.Create.
CREATE UNIQUE INDEX IF NOT EXISTS daily_problems_assigned_date_key
    ON daily_problems (assigned_date);

CREATE TABLE IF NOT EXISTS code_submissions (
    id               BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT      NOT NULL,
    daily_problem_id BIGINT      NOT NULL REFERENCES daily_problems (id) ON DELETE CASCADE,
    code             TEXT        NOT NULL,
    language         TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- CodeSubmissionRepository.Get uses QueryRow, so at most one submission may
-- exist per user and problem.
CREATE UNIQUE INDEX IF NOT EXISTS code_submissions_user_problem_key
    ON code_submissions (telegram_user_id, daily_problem_id);

CREATE TABLE IF NOT EXISTS telegram_problem_messages (
    id                  BIGSERIAL PRIMARY KEY,
    telegram_user_id    BIGINT      NOT NULL,
    daily_problem_id    BIGINT      NOT NULL REFERENCES daily_problems (id) ON DELETE CASCADE,
    telegram_message_id BIGINT      NOT NULL,
    sent_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One daily problem message per user and problem: this is what stops the
-- daily send from delivering the same problem twice.
CREATE UNIQUE INDEX IF NOT EXISTS telegram_problem_messages_user_problem_key
    ON telegram_problem_messages (telegram_user_id, daily_problem_id);

-- GetByMessageID resolves a reply to exactly one problem message.
CREATE UNIQUE INDEX IF NOT EXISTS telegram_problem_messages_user_message_key
    ON telegram_problem_messages (telegram_user_id, telegram_message_id);

-- GetLatestByUser orders by sent_at DESC, id DESC.
CREATE INDEX IF NOT EXISTS telegram_problem_messages_user_sent_at_idx
    ON telegram_problem_messages (telegram_user_id, sent_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS github_connection_states (
    state            TEXT PRIMARY KEY,
    telegram_user_id BIGINT      NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS github_connection_states_expires_at_idx
    ON github_connection_states (expires_at);

COMMIT;
