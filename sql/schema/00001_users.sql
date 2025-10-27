-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    user_id UUID NOT NULL PRIMARY KEY,
    username TEXT NOT NULL,
    email VARCHAR UNIQUE NOT NULL
);

CREATE TABLE passwords (
  user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  hashed_password VARCHAR NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users, passwords;
-- +goose StatementEnd
