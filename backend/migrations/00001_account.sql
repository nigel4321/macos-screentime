-- +goose Up
-- +goose StatementBegin
CREATE TABLE account (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE account_identity (
    provider    TEXT NOT NULL CHECK (provider IN ('apple', 'google')),
    subject_id  TEXT NOT NULL,
    account_id  UUID NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject_id)
);

CREATE INDEX idx_account_identity_account ON account_identity(account_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE account_identity;
DROP TABLE account;
-- +goose StatementEnd
