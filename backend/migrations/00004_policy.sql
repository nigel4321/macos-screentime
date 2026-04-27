-- +goose Up
-- +goose StatementBegin
CREATE TABLE policy (
    account_id  UUID NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    version     BIGINT NOT NULL,
    body_json   JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, version)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE policy;
-- +goose StatementEnd
