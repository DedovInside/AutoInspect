-- +migrate Down
DROP TABLE IF EXISTS auth_oauth_identities CASCADE;
