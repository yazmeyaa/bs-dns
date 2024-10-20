-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "dns_records" (
    id INTEGER PRIMARY KEY,
    label VARCHAR(255) NOT NULL,
    ipaddr VARCHAR(15) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(255)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE "dns_records";

-- +goose StatementEnd