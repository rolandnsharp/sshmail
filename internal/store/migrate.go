package store

import "database/sql"

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			fingerprint TEXT NOT NULL UNIQUE,
			public_key TEXT NOT NULL,
			bio TEXT NOT NULL DEFAULT '',
			public BOOLEAN NOT NULL DEFAULT 0,
			joined_at DATETIME NOT NULL DEFAULT (datetime('now')),
			invited_by INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_id INTEGER NOT NULL REFERENCES agents(id),
			to_id INTEGER NOT NULL REFERENCES agents(id),
			body TEXT NOT NULL,
			file_name TEXT,
			file_path TEXT,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			read_at DATETIME
		);

		CREATE INDEX IF NOT EXISTS idx_messages_to ON messages(to_id, read_at);
		CREATE INDEX IF NOT EXISTS idx_messages_from ON messages(from_id);

		CREATE TABLE IF NOT EXISTS group_members (
			group_id INTEGER NOT NULL REFERENCES agents(id),
			member_id INTEGER NOT NULL REFERENCES agents(id),
			role TEXT NOT NULL DEFAULT 'member',
			joined_at DATETIME NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (group_id, member_id)
		);

		CREATE TABLE IF NOT EXISTS agent_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id INTEGER NOT NULL REFERENCES agents(id),
			fingerprint TEXT NOT NULL UNIQUE,
			public_key TEXT NOT NULL,
			added_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS invites (
			code TEXT PRIMARY KEY,
			created_by INTEGER NOT NULL REFERENCES agents(id),
			redeemed_by INTEGER REFERENCES agents(id),
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			redeemed_at DATETIME
		);
	`)
	if err != nil {
		return err
	}
	// Rename legacy "board" agent to "general" (was auto-seeded, now channels are explicit)
	_, err = db.Exec(`UPDATE agents SET name = 'general', fingerprint = 'general' WHERE name = 'board' AND fingerprint = 'board'`)
	if err != nil {
		return err
	}

	// Migrate existing agent keys into agent_keys table
	_, err = db.Exec(`
		INSERT OR IGNORE INTO agent_keys (agent_id, fingerprint, public_key)
		SELECT id, fingerprint, public_key FROM agents
		WHERE public_key != '' AND fingerprint NOT LIKE 'group:%'
	`)
	if err != nil {
		return err
	}

	// Add email column to agents (ignore error if column already exists)
	db.Exec(`ALTER TABLE agents ADD COLUMN email TEXT DEFAULT NULL`)
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_agents_email ON agents(email) WHERE email IS NOT NULL`)
	if err != nil {
		return err
	}

	return migrateMessageReads(db)
}

// migrateMessageReads moves read state from the single messages.read_at column,
// which every reader of a message shared, to one row per reader. The old flags
// can't be trusted (a sender's message went "read" when the recipient opened it,
// and a group message when any member did), so on the first run every existing
// message counts as read by everyone who could see it. Guarded by user_version
// so the backfill runs once.
func migrateMessageReads(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS message_reads (
			message_id INTEGER NOT NULL REFERENCES messages(id),
			agent_id INTEGER NOT NULL REFERENCES agents(id),
			read_at DATETIME NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (message_id, agent_id)
		)`); err != nil {
		return err
	}

	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version >= 1 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		// Direct recipients.
		`INSERT OR IGNORE INTO message_reads (message_id, agent_id, read_at)
		 SELECT id, to_id, COALESCE(read_at, datetime('now')) FROM messages`,
		// Group members.
		`INSERT OR IGNORE INTO message_reads (message_id, agent_id, read_at)
		 SELECT m.id, gm.member_id, COALESCE(m.read_at, datetime('now'))
		 FROM messages m JOIN group_members gm ON gm.group_id = m.to_id`,
		`PRAGMA user_version = 1`,
	} {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	return tx.Commit()
}
