package store

import (
	"fmt"
	"testing"
)

func newTestStore(t *testing.T, dir string) *SQLiteStore {
	t.Helper()
	s, err := NewSQLite(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func mustAgent(t *testing.T, s *SQLiteStore, name string) *Agent {
	t.Helper()
	a, err := s.CreateAgent(name, "fp-"+name, "key-"+name, 0)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func mustSend(t *testing.T, s *SQLiteStore, from, to *Agent, body string) int64 {
	t.Helper()
	id, err := s.SendMessage(from.ID, to.ID, body, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// unread checks that poll's count, the unread inbox and the per-conversation
// counts all agree on n.
func unread(t *testing.T, s *SQLiteStore, a *Agent, n int) {
	t.Helper()
	count, err := s.UnreadCount(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := s.Inbox(a.ID, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	counts, err := s.UnreadCounts(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	sum := 0
	for _, c := range counts {
		sum += c
	}
	if count != n || len(msgs) != n || sum != n {
		t.Fatalf("%s: want %d unread, got poll=%d inbox=%d counts=%v", a.Name, n, count, len(msgs), counts)
	}
}

func TestOwnMessagesNeverUnread(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	alice := mustAgent(t, s, "alice")
	bob := mustAgent(t, s, "bob")
	group, err := s.CreateGroup("team", "", alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddGroupMember(group.ID, bob.ID); err != nil {
		t.Fatal(err)
	}

	mustSend(t, s, alice, bob, "dm")
	mustSend(t, s, alice, group, "group post")

	unread(t, s, alice, 0)
	unread(t, s, bob, 2)
}

func TestReadIsPerReader(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	alice := mustAgent(t, s, "alice")
	bob := mustAgent(t, s, "bob")
	carol := mustAgent(t, s, "carol")
	group, err := s.CreateGroup("team", "", alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []*Agent{bob, carol} {
		if err := s.AddGroupMember(group.ID, m.ID); err != nil {
			t.Fatal(err)
		}
	}

	post := mustSend(t, s, alice, group, "group post")
	unread(t, s, bob, 1)
	unread(t, s, carol, 1)

	// Bob reading the group post leaves it unread for Carol.
	if err := s.MarkRead(post, bob.ID); err != nil {
		t.Fatal(err)
	}
	unread(t, s, bob, 0)
	unread(t, s, carol, 1)

	// Reading twice is harmless.
	if err := s.MarkRead(post, bob.ID); err != nil {
		t.Fatal(err)
	}
	unread(t, s, bob, 0)

	// The inbox --all listing reports each reader's own read time.
	all, err := s.Inbox(carol.ID, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ReadAt != nil {
		t.Fatalf("carol: want the post listed unread, got %+v", all)
	}
	all, err = s.Inbox(bob.ID, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ReadAt == nil {
		t.Fatalf("bob: want the post listed read, got %+v", all)
	}
}

func TestUnreadCountsByConversation(t *testing.T) {
	s := newTestStore(t, t.TempDir())
	alice := mustAgent(t, s, "alice")
	bob := mustAgent(t, s, "bob")
	group, err := s.CreateGroup("team", "", alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddGroupMember(group.ID, bob.ID); err != nil {
		t.Fatal(err)
	}

	mustSend(t, s, alice, bob, "dm 1")
	mustSend(t, s, alice, bob, "dm 2")
	mustSend(t, s, alice, group, "group post")

	counts, err := s.UnreadCounts(bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Direct messages are keyed by the sender, group posts by the group.
	if counts["alice"] != 2 || counts["team"] != 1 || len(counts) != 2 {
		t.Fatalf("want map[alice:2 team:1], got %v", counts)
	}
}

func TestMigrationMarksExistingMessagesRead(t *testing.T) {
	dir := t.TempDir()
	s := newTestStore(t, dir)
	alice := mustAgent(t, s, "alice")
	bob := mustAgent(t, s, "bob")
	group, err := s.CreateGroup("team", "", alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddGroupMember(group.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		mustSend(t, s, alice, bob, fmt.Sprintf("dm %d", i))
		mustSend(t, s, bob, group, fmt.Sprintf("post %d", i))
	}
	// Simulate a database from before message_reads existed.
	for _, q := range []string{`DELETE FROM message_reads`, `PRAGMA user_version = 0`} {
		if _, err := s.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	unread(t, s, bob, 3)
	unread(t, s, alice, 3)
	s.Close()

	s = newTestStore(t, dir)
	unread(t, s, alice, 0)
	unread(t, s, bob, 0)

	// The backfill runs once: new mail after it is unread as usual.
	mustSend(t, s, alice, bob, "after")
	s.Close()
	s = newTestStore(t, dir)
	unread(t, s, bob, 1)
}
