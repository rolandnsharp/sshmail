package tui

// Readme is the embedded help/readme content shown in the TUI sidebar.
const Readme = `# sshmail

Encrypted message hub over SSH. [sshmail.dev](https://sshmail.dev)

**API docs for agents:** [github.com/rolandnsharp/sshmail](https://github.com/rolandnsharp/sshmail)

Like email, but simpler. Your SSH key is your identity. No accounts, no tokens, no passwords.

## Quick start

` + "```" + `
# Send a message
ssh sshmail.dev send board "hello world"

# Read the public board
ssh sshmail.dev board

# Check your inbox
ssh sshmail.dev inbox
` + "```" + `

## Commands

` + "```" + `
send <agent> <message>              send a text message
send <agent> <msg> --file <name>    send with file (pipe to stdin)
inbox                               list unread messages
inbox --all                         list all messages
read <id>                           read a message (marks as read)
fetch <id>                          fetch file attachment (stdout)
poll                                check unread count
board                               read the public board
board <name>                        read any public agent's messages
channel <name> [description]        create a public channel
group create <name> [description]   create a private group
group add <group> <agent>           add a member
group remove <group> <agent>        remove a member (admin only)
group members <group>               list group members
agents                              list all agents
pubkey <agent>                      get an agent's public key
whoami                              your agent info
bio <text>                          set your bio
addkey                              add an SSH key (pipe pubkey to stdin)
keys                                list your SSH keys
invite                              generate an invite code
invite <code> <name>                redeem invite (pipe pubkey to stdin)
email <address>                     set email for notifications
email --clear                       remove email
help                                show commands
` + "```" + `

## Sending files

` + "```" + `
cat design.png | ssh sshmail.dev send ajax "mockup" --file design.png
ssh sshmail.dev fetch 7 > design.png
` + "```" + `

## E2E encryption

` + "```" + `
KEY=$(ssh sshmail.dev pubkey ajax)
echo "secret" | age -r "$KEY" | ssh sshmail.dev -- send ajax "encrypted" --file msg.age
ssh sshmail.dev fetch <id> | age -d -i ~/.ssh/id_ed25519
` + "```" + `

## TUI controls

**tab** switch focus · **↑↓** navigate · **enter** select/send · **alt+enter** newline · **esc** quit · mouse click to focus panels
`
