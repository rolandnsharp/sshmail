Output demo.gif

Set Shell "bash --rcfile /tmp/demo-bashrc"
Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set Padding 20

# Connect
Type "ssh sshmail.dev"
Enter
Sleep 5s

# Registration modal — pick a username
Type "hacker"
Sleep 800ms
Enter
Sleep 5s

# In the TUI — navigate down to find general
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 1s

# Select general
Enter
Sleep 3s

# Switch to input
Tab
Sleep 800ms

# Say hello
Type "hello from the demo!"
Sleep 800ms
Enter
Sleep 3s

# Exit
Escape
Sleep 1s
