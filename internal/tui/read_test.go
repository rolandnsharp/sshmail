package tui

import (
	"reflect"
	"testing"
	"time"
)

// fakeBackend serves a fixed message list and records MarkRead calls.
type fakeBackend struct {
	msgs   []Message
	marked []int64
}

func (f *fakeBackend) Whoami() (*Agent, error)                  { return nil, nil }
func (f *fakeBackend) Agents() ([]Agent, error)                 { return nil, nil }
func (f *fakeBackend) Inbox(all bool) ([]Message, error)        { return f.msgs, nil }
func (f *fakeBackend) Board(name string) ([]Message, error)     { return nil, nil }
func (f *fakeBackend) Send(to, msg string) (*SendResult, error) { return nil, nil }
func (f *fakeBackend) PollCounts() (*PollResult, error)         { return &PollResult{}, nil }
func (f *fakeBackend) Watch(events chan<- WatchEvent) error     { return nil }
func (f *fakeBackend) Online() (map[string]bool, error)         { return nil, nil }
func (f *fakeBackend) MarkRead(ids []int64) error {
	f.marked = append(f.marked, ids...)
	now := time.Now()
	for _, id := range ids {
		for i := range f.msgs {
			if f.msgs[i].ID == id {
				f.msgs[i].ReadAt = &now
			}
		}
	}
	return nil
}

func TestOpeningConversationMarksItRead(t *testing.T) {
	read := time.Now()
	fb := &fakeBackend{msgs: []Message{
		{ID: 1, From: "bob", To: "alice"},                // unread DM from bob
		{ID: 2, From: "alice", To: "bob"},                // alice's own
		{ID: 3, From: "bob", To: "alice", ReadAt: &read}, // already read
		{ID: 4, From: "carol", To: "alice"},              // another conversation
		{ID: 5, From: "bob", To: "team"},                 // bob in a group: the DM view shows it too
		{ID: 6, From: "carol", To: "team"},               // unread group post
	}}
	m := NewModel(fb)
	m.me = &Agent{Name: "alice"}

	msg := m.fetchChannel(channelItem{name: "bob", kind: "dm"})()
	if in, ok := msg.(inboxMsg); !ok || !in.marked || len(in.messages) != 4 {
		t.Fatalf("dm: got %+v", msg)
	}
	// Whatever the view shows unread is marked: here bob's DM and his group post.
	if !reflect.DeepEqual(fb.marked, []int64{1, 5}) {
		t.Fatalf("dm: want [1 5] marked, got %v", fb.marked)
	}

	fb.marked = nil
	m.fetchChannel(channelItem{name: "team", kind: "group"})()
	if !reflect.DeepEqual(fb.marked, []int64{6}) {
		t.Fatalf("group: want [6] marked, got %v", fb.marked)
	}

	// The inbox view lists unread messages; it must not mark them.
	fb.marked = nil
	if in := m.fetchChannel(channelItem{kind: "inbox"})().(inboxMsg); in.marked || fb.marked != nil {
		t.Fatalf("inbox: want nothing marked, got %v", fb.marked)
	}
}
