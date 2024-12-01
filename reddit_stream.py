import curses
import asyncio
import asyncpraw
import aiohttp
import locale
from datetime import datetime, timezone
import os
from dotenv import load_dotenv
import time
from asyncprawcore import RequestException, ServerError, ResponseException
import re
from thread_finder import ThreadFinder, ThreadType, Thread
import textwrap
from typing import List, Dict

# Load environment variables from .env file
load_dotenv()

# Constants for rate limiting
REQUESTS_PER_MINUTE = 60
MIN_REQUEST_INTERVAL = 60 / REQUESTS_PER_MINUTE  # seconds
REFRESH_INTERVAL = 10  # seconds between comment updates
MAX_RETRIES = 3

class RedditStreamTUI:
    def __init__(self, stdscr, submission_url):
        self.stdscr = stdscr
        self.submission_url = submission_url
        self.comments = []
        self.last_request_time = 0
        self.update_dimensions()
        self.scroll_position = 0
        self.resize_event = asyncio.Event()
        self.last_status_message = ""
        self.running = True
        self.status_message = ""
        self.thread_title = ""
        self.last_status_update = time.time()
        self.status_update_interval = 2.0
        
        # Status bar symbols - Unicode box drawing characters
        self.status_left_border = "┌"
        self.status_right_border = "┐"
        self.status_bottom_left = "└"
        self.status_bottom_right = "┘"
        self.status_separator = "─"
        self.status_vertical = "│"
        self.arrow_char = ">"
        
        # Headers matching the curl command
        self.headers = {
            "accept": "*/*",
            "accept-language": "en-GB,en;q=0.9,en-US;q=0.8",
            "dnt": "1",
            "origin": "https://reddit-stream.com",
            "priority": "u=1, i",
            "referer": "https://reddit-stream.com/",
            "sec-ch-ua": '"Microsoft Edge";v="131", "Chromium";v="131", "Not_A Brand";v="24"',
            "sec-ch-ua-mobile": "?0",
            "sec-ch-ua-platform": '"Windows"',
            "sec-fetch-dest": "empty",
            "sec-fetch-mode": "cors",
            "sec-fetch-site": "cross-site",
            "user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0"
        }
        self.session = aiohttp.ClientSession(headers=self.headers)
        
        # Initialize colors - Monokai inspired
        curses.start_color()
        curses.use_default_colors()
        
        # Background color (dark grey)
        curses.init_pair(1, 141, -1)     # Purple for usernames
        curses.init_pair(2, 208, -1)     # Orange for timestamps
        curses.init_pair(3, 231, -1)     # White for regular text
        curses.init_pair(4, 197, -1)     # Pink for errors
        curses.init_pair(5, 148, -1)     # Green for arrows
        curses.init_pair(6, 81, -1)      # Blue for status bar
        
        # Set background
        self.stdscr.bkgd(' ', curses.color_pair(3))

    def update_dimensions(self):
        """Update the dimensions of the window"""
        self.max_height, self.max_width = self.stdscr.getmaxyx()
        self.content_height = self.max_height - 3  # Reserve space for status bar
        self.content_width = self.max_width - 2    # Reserve space for borders

    def display_status(self, message):
        """Update the status message"""
        current_time = time.time()
        # Only update status message if it's been more than status_update_interval seconds
        if current_time - self.last_status_update >= self.status_update_interval:
            self.status_message = message
            self.last_status_update = current_time
            self.draw_status_bar()

    def safe_addstr(self, y: int, x: int, text: str, color_pair=None):
        """Safely add a string to the screen with bounds checking"""
        try:
            # Don't try to write outside the window
            if y >= self.max_height or x >= self.max_width:
                return
                
            # Truncate string if it would go beyond screen width
            max_length = self.max_width - x
            if len(text) > max_length:
                text = text[:max_length]
            
            if color_pair is not None:
                self.stdscr.addstr(y, x, text, color_pair)
            else:
                self.stdscr.addstr(y, x, text)
        except curses.error:
            pass

    def draw_status_bar(self):
        """Draw the status bar at the top of the screen"""
        try:
            # Clear the status bar area
            for i in range(3):
                self.safe_addstr(i, 0, " " * self.max_width)
            
            # Get current timestamp
            current_time = datetime.now().strftime('%H:%M:%S')
            
            # Get thread title (last part of URL)
            thread_title = self.submission_url.split('/')[-2]
            if len(thread_title) > 40:  # Truncate if too long
                thread_title = thread_title[:37] + "..."
            
            # Format status components
            status_text = f" {thread_title} {self.status_vertical} Last update: {current_time} {self.status_vertical} Comments: {len(self.comments)} "
            
            # Calculate padding to center the status text
            total_width = self.max_width - 2  # Account for border chars
            padding = (total_width - len(status_text)) // 2
            padded_status = " " * padding + status_text + " " * (total_width - len(status_text) - padding)
            
            # Draw top border with corners
            top_border = f"{self.status_left_border}{self.status_separator * (self.max_width - 2)}{self.status_right_border}"
            self.safe_addstr(0, 0, top_border, curses.color_pair(6))
            
            # Draw status text
            self.safe_addstr(1, 0, f"{self.status_vertical}", curses.color_pair(6))
            self.safe_addstr(1, 1, padded_status, curses.color_pair(6))
            self.safe_addstr(1, self.max_width - 1, f"{self.status_vertical}", curses.color_pair(6))
            
            # Draw bottom border with corners
            bottom_border = f"{self.status_bottom_left}{self.status_separator * (self.max_width - 2)}{self.status_bottom_right}"
            self.safe_addstr(2, 0, bottom_border, curses.color_pair(6))
            
        except Exception as e:
            self.display_error(f"Status bar error: {str(e)}")

    def redraw_comments(self):
        """Redraw all comments"""
        try:
            # Clear the comment area
            for i in range(3, self.max_height):
                self.safe_addstr(i, 0, " " * self.max_width)
            
            if not self.comments:
                self.safe_addstr(3, 0, "No comments yet...", curses.color_pair(3))
                self.stdscr.refresh()
                return
            
            # Calculate total lines needed for all comments
            formatted_comments = []
            total_lines = 0
            
            # Sort comments by creation time (oldest first)
            sorted_comments = sorted(self.comments, key=lambda x: x.get('created_utc', 0))
            
            # Process comments oldest to newest
            for comment in sorted_comments:
                formatted = self.format_comment(comment, self.content_width)
                if formatted['valid']:
                    formatted_comments.append(formatted)
                    total_lines += len(formatted['body_lines']) + 1  # +1 for header
            
            # Calculate which comments to show based on scroll position
            visible_lines = 0
            visible_comments = []
            available_height = self.max_height - 4  # Account for status bar and bottom line
            
            # Take comments from the end to show newest at bottom
            for comment in reversed(formatted_comments[-self.content_height:]):
                lines_needed = len(comment['body_lines']) + 1  # header + content
                
                # Only add comment if we have space for header AND at least one line of content
                if visible_lines + lines_needed <= available_height and lines_needed > 1:
                    visible_comments.insert(0, comment)
                    visible_lines += lines_needed
                else:
                    break
            
            # Draw visible comments
            current_line = 3  # Start below status bar
            for idx, comment in enumerate(visible_comments):
                is_last_comment = idx == len(visible_comments) - 1
                
                # If this is the last comment and we're too close to the bottom,
                # skip it to prevent header-only display
                if is_last_comment and current_line >= self.max_height - 2:
                    break
                
                # Draw header with colors
                header = comment['header']
                author_end = comment['author_end']
                timestamp_start = comment['timestamp_start']
                
                # Author name in purple
                self.safe_addstr(current_line, 0, header[:author_end], curses.color_pair(1))
                # Separator in white
                self.safe_addstr(current_line, author_end, header[author_end:timestamp_start], curses.color_pair(3))
                # Timestamp in orange
                self.safe_addstr(current_line, timestamp_start, header[timestamp_start:], curses.color_pair(2))
                
                current_line += 1
                
                # Draw comment body
                for line_idx, line in enumerate(comment['body_lines']):
                    if current_line >= self.max_height - 1:
                        break
                    
                    # If this is the last comment and we only have space for one more line,
                    # make sure it's a content line
                    if is_last_comment and current_line == self.max_height - 1 and line_idx == 0:
                        if self.arrow_char in line:  # Line with arrow
                            arrow_pos = line.index(self.arrow_char)
                            self.safe_addstr(current_line, 0, line[:arrow_pos], curses.color_pair(3))
                            self.safe_addstr(current_line, arrow_pos, self.arrow_char, curses.color_pair(5))
                            self.safe_addstr(current_line, arrow_pos + 1, line[arrow_pos + 1:], curses.color_pair(3))
                    else:
                        if self.arrow_char in line:  # Line with arrow
                            arrow_pos = line.index(self.arrow_char)
                            self.safe_addstr(current_line, 0, line[:arrow_pos], curses.color_pair(3))
                            self.safe_addstr(current_line, arrow_pos, self.arrow_char, curses.color_pair(5))
                            self.safe_addstr(current_line, arrow_pos + 1, line[arrow_pos + 1:], curses.color_pair(3))
                        else:
                            self.safe_addstr(current_line, 0, line, curses.color_pair(3))
                    
                    current_line += 1
            
            self.stdscr.refresh()
            
        except Exception as e:
            self.display_error(f"Display error: {str(e)}")

    async def rate_limit_wait(self):
        """Ensure we don't exceed Reddit's rate limits"""
        now = time.time()
        time_since_last = now - self.last_request_time
        if time_since_last < MIN_REQUEST_INTERVAL:
            await asyncio.sleep(MIN_REQUEST_INTERVAL - time_since_last)
        self.last_request_time = time.time()

    def handle_resize(self):
        """Handle terminal resize events"""
        try:
            # Get new dimensions
            old_height = self.max_height
            old_width = self.max_width
            self.update_dimensions()
            
            # Clear the entire screen
            self.stdscr.clear()
            
            # Redraw everything
            self.draw_status_bar()
            self.redraw_comments()
            self.stdscr.refresh()
            
            # Signal resize completion
            self.resize_event.set()
            self.resize_event.clear()
            
        except Exception as e:
            self.display_error(f"Resize error: {str(e)}")

    def extract_submission_id(self, url: str) -> str:
        """Extract submission ID from various Reddit URL formats"""
        patterns = [
            r'reddit\.com/r/[^/]+/comments/([a-z0-9]+)',
            r'redd\.it/([a-z0-9]+)',
            r'/comments/([a-z0-9]+)',
            r'reddit\.com/comments/([a-z0-9]+)'
        ]
        
        for pattern in patterns:
            match = re.search(pattern, url)
            if match:
                return match.group(1)
        
        raise ValueError("Could not extract submission ID from URL")

    async def fetch_newest_comments(self, submission_id: str) -> List[Dict]:
        """Fetch newest comments using Reddit's JSON API"""
        try:
            url = f"https://www.reddit.com/comments/{submission_id}.json"
            params = {
                "sort": "new",
                "limit": 100,
                "raw_json": 1
            }
            
            self.display_status("Fetching new comments...")
            async with self.session.get(url, params=params) as response:
                if response.status == 200:
                    data = await response.json()
                    if len(data) >= 2:  # Should have 2 elements: post and comments
                        comments_data = data[1]["data"]["children"]
                        processed_comments = [comment["data"] for comment in comments_data if "data" in comment]
                        self.display_status(f"Fetched {len(processed_comments)} comments")
                        return processed_comments
                    self.display_status("No comments found in response")
                    return []
                else:
                    self.display_error(f"Error fetching comments: HTTP {response.status}")
                    return []
        except Exception as e:
            self.display_error(f"Error fetching comments: {str(e)}")
            return []

    def format_comment(self, comment: Dict, max_width: int) -> Dict:
        """Format a comment for display"""
        try:
            # Safely get comment data with defaults
            author = comment.get('author', '[deleted]')
            body = comment.get('body', '[deleted]')
            created_utc = comment.get('created_utc', time.time())
            
            # Clean up text to avoid display issues
            author = author.encode('ascii', 'replace').decode()
            body = body.encode('ascii', 'replace').decode()
            
            # Format timestamp
            timestamp = datetime.fromtimestamp(created_utc, tz=timezone.utc).strftime('%Y-%m-%d %H:%M:%S UTC')
            
            # Format header
            header = f"{author} | {timestamp}"
            
            # Format body
            body_lines = []
            words = body.split()
            current_line = []
            current_length = 2  # Account for arrow and space after
            
            for word in words:
                word_length = len(word)
                if current_length + word_length + 1 > max_width - 2:
                    # Line would be too long, start a new one
                    if current_line:
                        line = f"{self.arrow_char} {' '.join(current_line)}" if not body_lines else f"    {' '.join(current_line)}"
                        body_lines.append(line)
                    current_line = [word]
                    current_length = len(word) + 2
                else:
                    current_line.append(word)
                    current_length += word_length + 1
            
            # Add the last line if there is one
            if current_line:
                line = f"{self.arrow_char} {' '.join(current_line)}" if not body_lines else f"    {' '.join(current_line)}"
                body_lines.append(line)
            
            return {
                'valid': True,
                'header': header,
                'author_end': len(author),
                'timestamp_start': len(author) + 3,
                'body_lines': body_lines
            }
            
        except Exception as e:
            self.display_error(f"Error formatting comment: {str(e)}")
            return {'valid': False}

    def display_error(self, message):
        """Display an error message in the status bar"""
        self.display_status(f"ERROR: {message}")

    async def handle_input(self):
        """Handle user input"""
        while self.running:
            try:
                key = self.stdscr.getch()
                if key == ord('q'):
                    self.running = False
                    break
                elif key == curses.KEY_UP and self.scroll_position > 0:
                    self.scroll_position -= 1
                    self.redraw_comments()
                elif key == curses.KEY_DOWN:
                    self.scroll_position += 1
                    self.redraw_comments()
                elif key == curses.KEY_RESIZE:
                    self.resize_event.set()
            except Exception as e:
                self.display_error(f"Input error: {str(e)}")
            await asyncio.sleep(0.01)

    async def cleanup(self):
        """Cleanup resources"""
        try:
            # Clear the screen
            self.stdscr.clear()
            self.stdscr.refresh()
            
            # Reset terminal attributes
            curses.nocbreak()
            self.stdscr.keypad(False)
            curses.echo()
            curses.endwin()
            
            # Close the session
            if self.session and not self.session.closed:
                await self.session.close()
                
        except Exception as e:
            print(f"Error during cleanup: {str(e)}")

    async def fetch_comments(self):
        """Main comment fetching loop"""
        try:
            submission_id = self.extract_submission_id(self.submission_url)
            if not submission_id:
                self.display_error("Invalid Reddit URL")
                return

            self.display_status("Starting comment stream...")
            seen_ids = set()  # Track seen comment IDs
            
            while self.running:
                await self.rate_limit_wait()
                
                # Fetch new comments without status update
                new_comments = await self.fetch_newest_comments(submission_id)
                added_count = 0
                
                # Process new comments
                for comment in new_comments:
                    comment_id = comment.get('id')
                    if comment_id and comment_id not in seen_ids:
                        seen_ids.add(comment_id)
                        self.comments.append(comment)
                        added_count += 1
                
                # Sort comments by creation time (oldest first)
                if self.comments:
                    self.comments.sort(key=lambda x: x.get('created_utc', 0))
                    
                    # Keep only newest 1000 comments
                    if len(self.comments) > 1000:
                        self.comments = self.comments[-1000:]
                    
                    # Update display if we have new comments
                    if added_count > 0:
                        self.display_status(f"{len(self.comments)} comments")
                        self.redraw_comments()
                
                await asyncio.sleep(2)  # Wait before next fetch
                
        except Exception as e:
            self.display_error(f"Error in comment fetch loop: {str(e)}")

    async def run(self):
        """Main run method that coordinates all tasks"""
        try:
            # Set up initial screen
            curses.curs_set(0)  # Hide cursor
            self.stdscr.clear()
            self.stdscr.refresh()
            
            # Create tasks
            input_task = asyncio.create_task(self.handle_input())
            fetch_task = asyncio.create_task(self.fetch_comments())
            
            # Wait for tasks to complete
            await asyncio.gather(input_task, fetch_task)
            
        except Exception as e:
            self.display_error(f"Fatal error: {str(e)}")
        finally:
            # Ensure cleanup happens
            self.running = False
            await self.cleanup()

class ThreadSelector:
    def __init__(self, stdscr):
        self.stdscr = stdscr
        self.current_selection = 0
        
        # Load menu items from config and sort alphabetically by title
        config = ThreadType.load_config()
        self.thread_types = sorted(
            [(item["title"], item["type"]) for item in config.get("menu_items", [])],
            key=lambda x: x[0].lower()  # Sort case-insensitive
        )
        
        # Initialize colors
        curses.init_pair(1, curses.COLOR_CYAN, curses.COLOR_BLACK)    # Header
        curses.init_pair(2, curses.COLOR_WHITE, curses.COLOR_BLACK)   # Normal text
        curses.init_pair(3, curses.COLOR_BLACK, curses.COLOR_WHITE)   # Selected item
        
        # Initial screen setup
        curses.curs_set(0)  # Hide cursor
        self.stdscr.clear()
        self.stdscr.move(0, 0)  # Move cursor to home position

    def display(self):
        """Display thread type selection menu"""
        try:
            self.stdscr.clear()
            max_height, max_width = self.stdscr.getmaxyx()
            
            # Calculate center position
            menu_height = len(self.thread_types) + 2  # +2 for header and spacing
            start_y = (max_height - menu_height) // 2
            
            # Ensure start_y is never negative
            start_y = max(0, start_y)
            
            # Display header
            header = "Select Thread Type:"
            header_x = (max_width - len(header)) // 2
            self.stdscr.addstr(start_y, header_x, header, curses.color_pair(1))
            
            # Display options
            for idx, (thread_name, _) in enumerate(self.thread_types):
                y = start_y + 2 + idx  # +2 to leave space after header
                x = (max_width - len(thread_name)) // 2  # Center each option
                
                if idx == self.current_selection:
                    # Selected item
                    self.stdscr.attron(curses.color_pair(3))
                    self.stdscr.addstr(y, x, thread_name)
                    self.stdscr.attroff(curses.color_pair(3))
                else:
                    # Non-selected item
                    self.stdscr.addstr(y, x, thread_name, curses.color_pair(2))
            
            self.stdscr.refresh()
            
        except curses.error as e:
            pass  # Handle potential curses errors

    def handle_input(self, key):
        """Handle user input for thread selection"""
        if key == curses.KEY_UP and self.current_selection > 0:
            self.current_selection -= 1
            self.display()
        elif key == curses.KEY_DOWN and self.current_selection < len(self.thread_types) - 1:
            self.current_selection += 1
            self.display()
        elif key in [curses.KEY_ENTER, ord('\n'), ord(' ')]:
            return self.thread_types[self.current_selection][1]
        return None

    async def get_selection(self):
        """Get user's thread type selection"""
        self.display()
        while True:
            try:
                key = self.stdscr.getch()
                result = self.handle_input(key)
                if result is not None:
                    return result
                await asyncio.sleep(0.01)  # Prevent CPU hogging
            except Exception:
                continue

async def main(stdscr):
    # Set up the screen
    curses.start_color()
    curses.use_default_colors()
    curses.curs_set(0)  # Hide cursor
    stdscr.timeout(100)  # Non-blocking input
    stdscr.keypad(1)    # Enable keypad
    
    try:
        # Select thread
        selector = ThreadSelector(stdscr)
        thread_type = await selector.get_selection()
        
        # Find threads of selected type
        finder = ThreadFinder()
        try:
            threads = await finder.find_threads(thread_type)
        finally:
            await finder.close()
        
        if not threads:
            stdscr.addstr(0, 0, "No threads found! Press any key to exit.", curses.A_BOLD)
            stdscr.refresh()
            stdscr.getch()
            return
        
        # Now select specific thread
        selector = ThreadSelector(stdscr)
        selector.thread_types = [(thread.title, thread.url) for thread in threads]
        selector.current_selection = 0
        submission_url = await selector.get_selection()
        
        if not submission_url:
            return
        
        # Create and run TUI instance
        tui = RedditStreamTUI(stdscr, submission_url)
        await tui.run()
        
    except Exception as e:
        # Clear screen and display error
        stdscr.clear()
        stdscr.addstr(0, 0, f"Error: {str(e)}", curses.A_BOLD)
        stdscr.refresh()
        stdscr.getch()

if __name__ == "__main__":
    try:
        # Set up locale to handle Unicode
        locale.setlocale(locale.LC_ALL, '')
        curses.wrapper(lambda stdscr: asyncio.run(main(stdscr)))
    except KeyboardInterrupt:
        pass