Output demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set Padding 20

Env BASH_ENV "/tmp/demo-bashrc"
Env PS1 "$ "

# Connect
Type "ssh -i /tmp/testkey -o IdentitiesOnly=yes sshmail.dev"
Enter
Sleep 3s

# Registration modal — pick a username
Type "lol"
Sleep 500ms
Enter
Sleep 3s

# In the TUI — navigate down to find general
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 500ms

# Select general
Enter
Sleep 2s

# Switch to input
Tab
Sleep 500ms

# Say hello
Type "hello from the demo!"
Sleep 500ms
Enter
Sleep 2s

# Exit
Escape
Sleep 1s
