// sshmail TUI — Discord-like terminal client for sshmail
//
// Architecture:
//   - Connects to the hub via x/crypto/ssh (no shelling out)
//   - Three-panel layout: sidebar (groups/channels/DMs) | messages (viewport) | input (textarea)
//   - Polls the hub on a tick for real-time message updates
//   - Charm stack: Bubble Tea for the app loop, Bubbles for components, Lip Gloss for layout
//
// Components:
//   - Sidebar (bubbles/list): groups, public channels, DMs. Tab to toggle focus.
//   - Messages (bubbles/viewport): scrollable message history, auto-scroll on new messages.
//   - Input (bubbles/textarea): multi-line compose. Enter to send.
//
// Layout (Lip Gloss):
//   ┌──────────────┬───────────────────────────────┐
//   │ # devs       │ ajax: hey team                │
//   │ # sshmail    │ roland: shipped groups         │
//   │   board      │ lisa: nice, adding webhooks    │
//   │   blah       │                               │
//   │              │                               │
//   │ DMs          ├───────────────────────────────┤
//   │   ajax       │ > type a message...           │
//   │   russell    │                               │
//   └──────────────┴───────────────────────────────┘
//
//   Sidebar ~35% width. Messages fill the rest minus input height.
//   lipgloss.JoinHorizontal(sidebar, lipgloss.JoinVertical(messages, input))
//
// State:
//   - Focus: sidebar or input (Tab to toggle)
//   - Current channel/DM selection drives which messages are shown
//   - Unread counts shown in sidebar
//
// Data flow:
//   - tea.Tick every 5s → ssh poll → update unread counts
//   - Selecting a channel → ssh inbox/board → populate viewport
//   - Enter in input → ssh send → append to viewport
//   - All hub commands return JSON, parsed into Go structs (reuse internal/store types)
//
// Config:
//   - SSHMAIL_HOST (default: ssh.sshmail.dev)
//   - SSHMAIL_PORT (default: 2233)
//   - SSHMAIL_KEY  (default: ~/.ssh/id_ed25519)
//
// Pattern: "Smart Model, Dumb Components" (from Charmbracelet/Crush)
//   - Single root model handles all tea.Msg routing
//   - Components expose SetSize(), Focus(), Blur() — not full Update cycles
//   - Focus state determines keyboard routing
//   - Message priority: resize → poll updates → keyboard → paste

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("sshmail tui — not yet implemented")
	fmt.Println("see cmd/tui/main.go for architecture notes")
	os.Exit(0)
}
