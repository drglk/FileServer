CREATE TABLE IF NOT EXISTS grants (
			document_id UUID,
			user_id UUID,
			FOREIGN KEY(document_id) REFERENCES documents(id),
			FOREIGN KEY(user_id) REFERENCES users(id),
			PRIMARY KEY (document_id, user_id)
		);