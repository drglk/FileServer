CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY,
    login Varchar(255) NOT NULL UNIQUE,
    pass_hash BYTEA NOT NULL
    );