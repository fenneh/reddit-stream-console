# Reddit Stream Console

A terminal-based Reddit thread viewer built with [Textual](https://github.com/Textualize/textual). Watch live comments from Reddit threads in your terminal with a beautiful TUI interface.

![Reddit Stream Console Screenshot](docs/screenshot.png)

## Features

- Live comment streaming from Reddit threads
- Support for multiple thread types:
  - Soccer match threads
  - Soccer post-match threads
  - Fantasy Premier League rant threads
  - NFL game threads
  - NFL post-game threads
- Beautiful terminal UI with:
  - Easy navigation with keyboard or mouse
  - Comment threading and indentation
  - Color-coded usernames, scores, and timestamps
  - Auto-scrolling with new comments
  - Configurable refresh rate

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/reddit-stream-console.git
cd reddit-stream-console
```

2. Create a virtual environment and install dependencies:
```bash
python -m venv venv
source venv/bin/activate  # On Windows: .\venv\Scripts\activate
pip install -r requirements.txt
```

3. Set up your Reddit API credentials:
   - Create a Reddit app at https://www.reddit.com/prefs/apps/
   - Create a `.env` file with your credentials:
```env
REDDIT_CLIENT_ID=your_client_id
REDDIT_CLIENT_SECRET=your_client_secret
REDDIT_USER_AGENT=your_user_agent
```

## Usage

Run the app:
```bash
# Make sure your virtual environment is activated
python main.py
```

Navigation:
- Use arrow keys or mouse to navigate menus
- Enter to select
- Escape to go back
- Q to quit
- R to manually refresh comments
- End to scroll to bottom

## Configuration

Thread types and search criteria can be configured in `config/menu_config.json`. Each thread type can specify:
- Subreddit to search in
- Required title keywords
- Required flair
- Maximum thread age
- Number of threads to fetch

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
