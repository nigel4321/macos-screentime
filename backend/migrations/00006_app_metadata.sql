-- +goose Up
-- +goose StatementBegin
-- Per-account display-name catalog. The Mac agent (or any future
-- client that knows app metadata) upserts (bundle_id → display_name)
-- here; GET /v1/usage:summary LEFT JOINs to surface human names to
-- clients that can't resolve them locally (Android, web, iOS).
--
-- Latest write wins — a renamed app overrides the previous name on
-- the next upsert. We don't keep history; a simple updated_at lets
-- callers reason about staleness if they ever need to.
CREATE TABLE app_metadata (
    account_id    UUID        NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    bundle_id     TEXT        NOT NULL,
    display_name  TEXT        NOT NULL CHECK (display_name <> ''),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, bundle_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE app_metadata;
-- +goose StatementEnd
