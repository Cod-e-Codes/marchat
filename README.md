# marchat 🧃

[![Go Version](https://img.shields.io/badge/go-1.18%2B-blue?logo=go)](https://go.dev/dl/)
[![MIT License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![GitHub Repo](https://img.shields.io/badge/github-repo-blue?logo=github)](https://github.com/Cod-e-Codes/marchat)

A modern, retro-inspired terminal chat app for father-son coding sessions. Built with Go, Bubble Tea, and SQLite (pure Go driver, no C compiler required). Fast, hackable, and ready for remote pair programming.

---

## Features

- **Terminal UI**: Beautiful, scrollable chat using [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Go WebSocket Server**: Real-time, robust, and cross-platform
- **SQLite (pure Go)**: No C compiler needed (uses `modernc.org/sqlite`)
- **Usernames & Timestamps**: See who said what, and when
- **Color Themes**: Slack, Discord, AIM, or classic
- **Emoji Support**: ASCII emoji auto-conversion
- **Configurable**: Set username, server URL, and theme via config or flags
- **Easy Quit**: Press `q` or `ctrl+c` to exit the chat

---

## Quick Start

### 1. Clone the repo
```sh
git clone https://github.com/Cod-e-Codes/marchat.git
cd marchat
```

### 2. Install Go dependencies
```sh
go mod tidy
```

### 3. Run the server (port 9090, WebSocket)
```sh
go run cmd/server/main.go
```

### 4. (Optional) Create a config file
Create `config.json` in the project root:
```json
{
  "username": "Cody",
  "server_url": "ws://localhost:9090/ws",
  "theme": "slack"
}
```

### 5. Run the client
```sh
# With flags:
go run client/main.go --username Cody --theme slack --server ws://localhost:9090/ws

# Or with config file:
go run client/main.go --config config.json
```

---

## Remote Usage

To connect from another machine (e.g. over the internet):
1. **Open port 9090** on your firewall/router and forward it to your server's local IP.
2. **Get your public IP** (e.g. from https://whatismyip.com or `curl ifconfig.me`).
3. **Run the server** on your host machine.
4. **Connect the client** using:
   ```sh
   go run client/main.go --username Dad --server ws://YOUR_PUBLIC_IP:9090/ws
   ```
   Or set `"server_url": "ws://YOUR_PUBLIC_IP:9090/ws"` in your config file.

If you can't port forward, use [ngrok](https://ngrok.com/):
```sh
ngrok http 9090
# Use the wss://.../ws URL provided by ngrok
```

---

## Usage
- **Send messages**: Type and press Enter
- **Quit**: Press `ctrl+c` or `Esc`
- **Themes**: `slack`, `discord`, `aim`, or leave blank for default
- **Emoji**: `:), :(, :D, <3, :P` auto-convert to Unicode
- **Scroll**: Use Up/Down arrows or your mouse to scroll chat
- **Switch theme**: Type `:theme <name>` and press Enter
- **Clear chat (client only)**: Type `:clear` and press Enter
- **Clear all messages (wipe DB)**: Type `:cleardb` and press Enter (removes all messages for everyone)
- **Banner**: Status and error messages appear above chat

---

## Project Structure
```
marchat/
├── client/           # TUI client (Bubble Tea)
│   ├── main.go
│   └── config/config.go
├── cmd/server/       # Server entrypoint
│   └── main.go
├── server/           # Server logic (DB, handlers, WebSocket)
│   ├── db.go
│   ├── handlers.go
│   ├── client.go
│   ├── hub.go
│   └── schema.sql
├── shared/           # Shared types
│   └── types.go
├── go.mod
├── go.sum
└── README.md
```

---

## Tech Stack
- [Go](https://golang.org/) 1.18+
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) (TUI)
- [Lipgloss](https://github.com/charmbracelet/lipgloss) (styling)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go SQLite)
- [Gorilla WebSocket](https://github.com/gorilla/websocket) (real-time messaging)

---

## Next Steps
- [ ] Persistent config file
- [ ] Avatars and richer themes
- [x] WebSocket support
- [ ] Deploy to cloud (Fly.io, AWS, etc.)

---

## Contributing
See [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

---

## License

This project is licensed under the [MIT License](LICENSE).
