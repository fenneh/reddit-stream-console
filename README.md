# reddit-stream-console

[![build](https://github.com/fenneh/reddit-stream-console/actions/workflows/build.yml/badge.svg)](https://github.com/fenneh/reddit-stream-console/actions/workflows/build.yml)
[![release](https://img.shields.io/github/v/release/fenneh/reddit-stream-console)](https://github.com/fenneh/reddit-stream-console/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![license](https://img.shields.io/github/license/fenneh/reddit-stream-console)](LICENSE)

Terminal-based Reddit comment streamer built with Go and [tview](https://github.com/rivo/tview). Like reddit-stream.com, but in your terminal.

![Screenshot](docs/screenshot.png)

## Download

Grab the latest binary for your platform from [Releases](https://github.com/fenneh/reddit-stream-console/releases).

No Reddit API credentials required.

## Features

- Real-time comment streaming with auto-refresh
- Live comment filtering
- Threaded comment display
- Keyboard-driven interface

## Building from Source

```bash
cd go
go build -o bin/reddit-stream-console ./cmd/reddit-stream-console
./bin/reddit-stream-console
```

## Controls

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate |
| `Enter` | Select |
| `/` | Filter comments |
| `r` | Refresh comments |
| `Esc` | Go back |
| `q` | Quit |

## Configuration

The app works out of the box with sensible defaults (soccer, NFL, and FantasyPL match threads). To customize the menu, create a `config/menu_config.json` file.

Config file search order:
1. `~/.reddit-stream-console/config/menu_config.json` (home directory)
2. Next to the executable
3. One directory above the executable
4. Two directories above the executable

If no config file is found, built-in defaults are used.

See `config/menu_config.json` for an example configuration.

## License

MIT
