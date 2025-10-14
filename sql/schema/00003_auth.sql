-- +goose Up
-- +goose StatementBegin
CREATE TABLE passwords (
  user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  hashed_password VARCHAR NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE passwords;
-- +goose StatementEnd
