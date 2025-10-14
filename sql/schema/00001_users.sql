-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    user_id UUID NOT NULL PRIMARY KEY,
    username TEXT NOT NULL,
    email VARCHAR NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
