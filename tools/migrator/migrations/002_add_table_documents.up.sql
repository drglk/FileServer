CREATE TABLE IF NOT EXISTS documents (
		id UUID PRIMARY KEY,
        owner_id UUID NOT NULL,
        name TEXT NOT NULL,
        mime TEXT NOT NULL,
        is_file BOOLEAN,
        is_public BOOLEAN,
        path TEXT,
        json_data JSONB,
        created_at TIMESTAMP,
        FOREIGN KEY(owner_id) REFERENCES users(id)
        );