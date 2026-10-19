package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLite(dataDir string) (*SQLiteStore, error) {
	dbPath := filepath.Join(dataDir, "hub.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

// --- Agents ---

func (s *SQLiteStore) AgentByFingerprint(fingerprint string) (*Agent, error) {
	// Look up via agent_keys first (supports multi-key)
	a := &Agent{}
	err := s.db.QueryRow(`
		SELECT a.id, a.name, a.fingerprint, a.public_key, a.bio, a.email, a.public, a.joined_at, a.invited_by
		FROM agents a
		JOIN agent_keys ak ON a.id = ak.agent_id
		WHERE ak.fingerprint = ?`, fingerprint).Scan(&a.ID, &a.Name, &a.Fingerprint, &a.PublicKey, &a.Bio, &a.Email, &a.Public, &a.JoinedAt, &a.InvitedBy)
	if err == sql.ErrNoRows {
		// Fall back to agents table for channels/groups/board
		return s.scanAgent(`SELECT id, name, fingerprint, public_key, bio, email, public, joined_at, invited_by FROM agents WHERE fingerprint = ?`, fingerprint)
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *SQLiteStore) AgentByID(id int64) (*Agent, error) {
	return s.scanAgent(`SELECT id, name, fingerprint, public_key, bio, email, public, joined_at, invited_by FROM agents WHERE id = ?`, id)
}

func (s *SQLiteStore) AgentByName(name string) (*Agent, error) {
	return s.scanAgent(`SELECT id, name, fingerprint, public_key, bio, email, public, joined_at, invited_by FROM agents WHERE name = ?`, name)
}

func (s *SQLiteStore) AgentByEmail(email string) (*Agent, error) {
	return s.scanAgent(`SELECT id, name, fingerprint, public_key, bio, email, public, joined_at, invited_by FROM agents WHERE email = ?`, email)
}

func (s *SQLiteStore) scanAgent(query string, arg any) (*Agent, error) {
	a := &Agent{}
	err := s.db.QueryRow(query, arg).Scan(&a.ID, &a.Name, &a.Fingerprint, &a.PublicKey, &a.Bio, &a.Email, &a.Public, &a.JoinedAt, &a.InvitedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

func (s *SQLiteStore) CreateAgent(name, fingerprint, publicKey string, invitedBy int64) (*Agent, error) {
	res, err := s.db.Exec(
		`INSERT INTO agents (name, fingerprint, public_key, invited_by) VALUES (?, ?, ?, ?)`,
		name, fingerprint, publicKey, invitedBy,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.AgentByID(id)
}

func (s *SQLiteStore) CreateChannel(name, bio string) (*Agent, error) {
	res, err := s.db.Exec(
		`INSERT INTO agents (name, fingerprint, public_key, bio, public) VALUES (?, ?, '', ?, 1)`,
		name, name, bio,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.AgentByID(id)
}

func (s *SQLiteStore) UpdateBio(id int64, bio string) error {
	_, err := s.db.Exec(`UPDATE agents SET bio = ? WHERE id = ?`, bio, id)
	return err
}

func (s *SQLiteStore) UpdateEmail(id int64, email *string) error {
	_, err := s.db.Exec(`UPDATE agents SET email = ? WHERE id = ?`, email, id)
	return err
}

func (s *SQLiteStore) ListAgents() ([]Agent, error) {
	rows, err := s.db.Query(`SELECT id, name, fingerprint, public_key, bio, email, public, joined_at, invited_by FROM agents ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.Name, &a.Fingerprint, &a.PublicKey, &a.Bio, &a.Email, &a.Public, &a.JoinedAt, &a.InvitedBy); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// --- Messages ---

func (s *SQLiteStore) SendMessage(fromID, toID int64, body string, fileName, filePath *string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO messages (from_id, to_id, body, file_name, file_path) VALUES (?, ?, ?, ?, ?)`,
		fromID, toID, body, fileName, filePath,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// unreadFor is true for a message the agent (bound twice, as the first two
// parameters) has not read. Read state is per reader, in message_reads, so one
// agent reading a message never marks it read for the sender or for the other
// members of a group, and an agent's own messages are never unread for them.
const unreadFor = `m.from_id != ? AND NOT EXISTS (
	SELECT 1 FROM message_reads r WHERE r.message_id = m.id AND r.agent_id = ?)`

// toAgent is true for a message addressed to the agent (bound three times):
// directly, to a group they belong to, or to a public board since they joined
// (so a new agent doesn't start with the boards' whole history unread).
const toAgent = `(m.to_id = ?
	OR m.to_id IN (SELECT group_id FROM group_members WHERE member_id = ?)
	OR (m.to_id IN (SELECT id FROM agents WHERE public)
		AND m.created_at >= (SELECT joined_at FROM agents WHERE id = ?)))`

func (s *SQLiteStore) Inbox(agentID int64, all bool, after *time.Time) ([]Message, error) {
	var query string
	var args []any
	if all {
		// Everything the agent sent or received; read_at is when they read it.
		query = `
		SELECT m.id, m.from_id, f.name, m.to_id, t.name, m.body, m.file_name, m.file_path, m.created_at, r.read_at
		FROM messages m
		JOIN agents f ON m.from_id = f.id
		JOIN agents t ON m.to_id = t.id
		LEFT JOIN message_reads r ON r.message_id = m.id AND r.agent_id = ?
		WHERE (m.from_id = ? OR ` + toAgent + `)`
		args = []any{agentID, agentID, agentID, agentID, agentID}
	} else {
		// Unread: the same set poll counts.
		query = `
		SELECT m.id, m.from_id, f.name, m.to_id, t.name, m.body, m.file_name, m.file_path, m.created_at, NULL
		FROM messages m
		JOIN agents f ON m.from_id = f.id
		JOIN agents t ON m.to_id = t.id
		WHERE ` + toAgent + ` AND ` + unreadFor
		args = []any{agentID, agentID, agentID, agentID, agentID}
	}
	if after != nil {
		query += ` AND m.created_at > ?`
		args = append(args, after.UTC().Format("2006-01-02 15:04:05"))
	}
	// Oldest first, so a listing reads like the conversation did. created_at
	// is whole seconds, so id breaks ties within a second.
	query += ` ORDER BY m.created_at, m.id`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.FromID, &m.FromName, &m.ToID, &m.ToName, &m.Body, &m.FileName, &m.FilePath, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// GetMessage has no reader, so it carries no read state (ReadAt is nil).
func (s *SQLiteStore) GetMessage(id int64) (*Message, error) {
	m := &Message{}
	err := s.db.QueryRow(`
		SELECT m.id, m.from_id, f.name, m.to_id, t.name, m.body, m.file_name, m.file_path, m.created_at, NULL
		FROM messages m
		JOIN agents f ON m.from_id = f.id
		JOIN agents t ON m.to_id = t.id
		WHERE m.id = ?
	`, id).Scan(&m.ID, &m.FromID, &m.FromName, &m.ToID, &m.ToName, &m.Body, &m.FileName, &m.FilePath, &m.CreatedAt, &m.ReadAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// MarkRead records that agentID has read the message. Other readers are unaffected.
func (s *SQLiteStore) MarkRead(messageID, agentID int64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO message_reads (message_id, agent_id) VALUES (?, ?)`, messageID, agentID)
	return err
}

func (s *SQLiteStore) UnreadCount(agentID int64) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM messages m
		WHERE `+toAgent+` AND `+unreadFor,
		agentID, agentID, agentID, agentID, agentID).Scan(&count)
	return count, err
}

// UnreadCounts is UnreadCount broken down by conversation: the group or board
// name for posts, the sender's name for direct messages.
func (s *SQLiteStore) UnreadCounts(agentID int64) (map[string]int, error) {
	rows, err := s.db.Query(`
		SELECT CASE WHEN m.to_id = ? THEN f.name ELSE t.name END AS conv, COUNT(*)
		FROM messages m
		JOIN agents f ON m.from_id = f.id
		JOIN agents t ON m.to_id = t.id
		WHERE `+toAgent+` AND `+unreadFor+`
		GROUP BY conv`, agentID, agentID, agentID, agentID, agentID, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		counts[name] = count
	}
	return counts, rows.Err()
}

// --- Keys ---

func (s *SQLiteStore) AddKey(agentID int64, fingerprint, publicKey string) error {
	_, err := s.db.Exec(
		`INSERT INTO agent_keys (agent_id, fingerprint, public_key) VALUES (?, ?, ?)`,
		agentID, fingerprint, publicKey,
	)
	return err
}

func (s *SQLiteStore) RemoveKey(agentID int64, fingerprint string) error {
	// Don't allow removing the last key
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM agent_keys WHERE agent_id = ?`, agentID).Scan(&count); err != nil {
		return err
	}
	if count <= 1 {
		return fmt.Errorf("cannot remove last key")
	}
	_, err := s.db.Exec(
		`DELETE FROM agent_keys WHERE agent_id = ? AND fingerprint = ?`,
		agentID, fingerprint,
	)
	return err
}

func (s *SQLiteStore) ListKeys(agentID int64) ([]AgentKey, error) {
	rows, err := s.db.Query(`SELECT fingerprint, public_key, added_at FROM agent_keys WHERE agent_id = ? ORDER BY added_at`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []AgentKey
	for rows.Next() {
		var k AgentKey
		if err := rows.Scan(&k.Fingerprint, &k.PublicKey, &k.AddedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// --- Groups ---

func (s *SQLiteStore) CreateGroup(name, bio string, adminID int64) (*Agent, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO agents (name, fingerprint, public_key, bio, public) VALUES (?, ?, '', ?, 0)`,
		name, "group:"+name, bio,
	)
	if err != nil {
		return nil, err
	}
	groupID, _ := res.LastInsertId()

	_, err = tx.Exec(
		`INSERT INTO group_members (group_id, member_id, role) VALUES (?, ?, 'admin')`,
		groupID, adminID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.AgentByID(groupID)
}

func (s *SQLiteStore) AddGroupMember(groupID, memberID int64) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO group_members (group_id, member_id, role) VALUES (?, ?, 'member')`,
		groupID, memberID,
	)
	return err
}

func (s *SQLiteStore) RemoveGroupMember(groupID, memberID int64) error {
	_, err := s.db.Exec(
		`DELETE FROM group_members WHERE group_id = ? AND member_id = ?`,
		groupID, memberID,
	)
	return err
}

func (s *SQLiteStore) GroupMembers(groupID int64) ([]GroupMember, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.name, gm.role, gm.joined_at
		FROM group_members gm
		JOIN agents a ON gm.member_id = a.id
		WHERE gm.group_id = ?
		ORDER BY gm.joined_at`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []GroupMember
	for rows.Next() {
		var m GroupMember
		if err := rows.Scan(&m.AgentID, &m.Name, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *SQLiteStore) GroupRole(groupID, agentID int64) (string, error) {
	var role string
	err := s.db.QueryRow(
		`SELECT role FROM group_members WHERE group_id = ? AND member_id = ?`,
		groupID, agentID,
	).Scan(&role)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return role, err
}

func (s *SQLiteStore) IsGroupMember(groupID, agentID int64) (bool, error) {
	role, err := s.GroupRole(groupID, agentID)
	if err != nil {
		return false, err
	}
	return role != "", nil
}

// --- Invites ---

func (s *SQLiteStore) CreateInvite(createdBy int64) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := hex.EncodeToString(b)
	_, err := s.db.Exec(`INSERT INTO invites (code, created_by) VALUES (?, ?)`, code, createdBy)
	return code, err
}

func (s *SQLiteStore) RedeemInvite(code, name, fingerprint, publicKey string) (*Agent, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var createdBy int64
	var redeemedBy sql.NullInt64
	err = tx.QueryRow(`SELECT created_by, redeemed_by FROM invites WHERE code = ?`, code).Scan(&createdBy, &redeemedBy)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid invite code")
	}
	if err != nil {
		return nil, err
	}
	if redeemedBy.Valid {
		return nil, fmt.Errorf("invite already redeemed")
	}

	res, err := tx.Exec(
		`INSERT INTO agents (name, fingerprint, public_key, invited_by) VALUES (?, ?, ?, ?)`,
		name, fingerprint, publicKey, createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	agentID, _ := res.LastInsertId()

	_, err = tx.Exec(
		`INSERT INTO agent_keys (agent_id, fingerprint, public_key) VALUES (?, ?, ?)`,
		agentID, fingerprint, publicKey,
	)
	if err != nil {
		return nil, fmt.Errorf("add key: %w", err)
	}

	_, err = tx.Exec(`UPDATE invites SET redeemed_by = ?, redeemed_at = datetime('now') WHERE code = ?`, agentID, code)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.AgentByID(agentID)
}
