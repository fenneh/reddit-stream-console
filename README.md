# Reddit Stream Console 🎭

Because sometimes you want to read Reddit like it's 1989.

![Screenshot](docs/screenshot.png)

## What is this?

A TUI (Text User Interface) application for streaming Reddit comments in real-time. Perfect for those moments when you want to:
- Pretend you're a hacker while reading about cats
- Follow live sports discussions without explaining to your boss why Reddit is open
- Experience the joy of ASCII art in comment threads
- Actually get work done (results may vary)

Inspired by [reddit-stream.com](https://reddit-stream.com), but for those times when you want your Reddit in glorious ASCII. Think of it as reddit-stream's terminal-dwelling cousin who took a different path in life.

## Features

- 🔄 Real-time comment streaming
- 🎨 Beautiful TUI interface (as beautiful as ASCII can be)
- 🔍 Live comment filtering (for when you're looking for that one comment that agrees with you)
- 📜 Auto-scrolling with manual override (because sometimes you need to pause the madness)
- 🎮 Keyboard-driven interface (mouse users, we still love you)
- 🌈 Color-coded comments (because monochrome is so 1988)

## Quick Start

### The "I Read Documentation" Way

1. Clone this repository:
   ```bash
   git clone https://github.com/yourusername/reddit-stream-console.git
   cd reddit-stream-console
   ```

2. Run the build script:

   **On Linux/macOS:**
   ```bash
   chmod +x build.sh
   ./build.sh
   ```

   **On Windows (PowerShell):**
   ```powershell
   .\build.ps1
   ```

3. Edit `.env` with your Reddit API credentials (get them at https://www.reddit.com/prefs/apps)

4. Run the app:
   ```bash
   # Linux/macOS
   source venv/bin/activate
   python main.py

   # Windows (PowerShell)
   .\venv\Scripts\Activate.ps1
   python main.py
   ```

### The "I Love Containers" Way

1. Build the Docker image:
   ```bash
   docker build -t reddit-stream-console .
   ```

2. Run it:
   ```bash
   docker run -it --env-file .env reddit-stream-console
   ```

## Controls

- `/` - Filter comments (because ctrl+f was too mainstream)
- `r` - Refresh comments manually (for the impatient)
- `end` - Scroll to bottom (in case you're lost in the void)
- `escape` - Show menu/exit filter (escape reality)
- `backspace` - Go back (time travel not guaranteed)
- `q` - Quit (when you've had enough internet for one day)

## Configuration

Copy `.env.example` to `.env` and fill in your Reddit API credentials. If you don't know how to get these:

1. Go to https://www.reddit.com/prefs/apps
2. Create a new application
3. Select "script"
4. Fill in the required fields (redirect URI can be http://localhost)
5. Get your client ID and secret
6. Question your life choices that led you to reading Reddit in a terminal

## Contributing

Found a bug? Want to add a feature? Have a existential crisis about terminal-based social media? Feel free to:

1. Open an issue
2. Submit a PR
3. Fork and create your own version (we won't judge)

## License

MIT - Because sharing is caring, and lawyers are expensive.

## Acknowledgments

- [reddit-stream.com](https://reddit-stream.com) for pioneering the idea of live Reddit comment streaming
- The Textual framework, for making TUI development less painful
- Reddit's API, for being relatively stable
- Coffee, for obvious reasons
- You, for reading this far (seriously, impressive commitment)
