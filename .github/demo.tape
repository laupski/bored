# VHS Demo Tape for bored TUI
# https://github.com/charmbracelet/vhs

Output demo.gif
Output demo.mp4

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"

# Set up fake credentials for demo (won't actually connect)
Set TypingSpeed 50ms

# Show the app starting
Type "./bin/bored"
Enter
Sleep 2s

# The app will show the config screen since no credentials are saved
# Navigate through the UI to show it works
Sleep 1s

# Type some sample config values (demo only)
Type "demo-org"
Sleep 500ms
Tab
Type "demo-project"
Sleep 500ms
Tab
Type "demo-team"
Sleep 500ms
Tab
Type "Demo\\Team"
Sleep 500ms
Tab
Type "demo-pat-token"
Sleep 500ms
Tab
Type "demo-user"
Sleep 500ms

# Show the filled form
Sleep 2s

# Exit gracefully
Escape
Sleep 500ms
Ctrl+C
Sleep 1s
