# reddit-stream-console

Terminal-based Reddit comment streamer. Like reddit-stream.com, but for people who prefer their social media in ASCII.

![Screenshot](docs/screenshot.png)

## Features

- Real-time comment streaming
- Live comment filtering
- Auto-scrolling with manual override
- Keyboard-driven interface
- Color-coded comments

## Quick Start

```bash
git clone https://github.com/fenneh/reddit-stream-console.git
cd reddit-stream-console
./build.sh  # or build.ps1 on Windows
```

Edit `.env` with your Reddit API credentials from https://www.reddit.com/prefs/apps

```bash
source venv/bin/activate
python main.py
```

## Docker

```bash
docker build -t reddit-stream-console .
docker run -it --env-file .env reddit-stream-console
```

## Controls

| Key | Action |
|-----|--------|
| `/` | Filter comments |
| `r` | Refresh |
| `end` | Scroll to bottom |
| `escape` | Menu/exit filter |
| `backspace` | Go back |
| `q` | Quit |

## License

MIT
