from textual.app import App, ComposeResult
from textual.widgets import Header, Footer, Label, Button
from textual.css.query import NoMatches
from textual.binding import Binding
import os
from dotenv import load_dotenv
import logging

from .components.menu_screen import MenuScreen
from .components.thread_list import ThreadListScreen
from .components.comment_viewer import CommentContainer, CommentWidget
from .components.url_input_screen import UrlInputScreen
from .services.reddit_service import RedditService
from .services.thread_finder import ThreadFinder
from .utils.config import load_menu_config

# Load environment variables
load_dotenv()

# Add logging configuration after load_dotenv()
logging.basicConfig(
    filename='reddit_stream_debug.log',
    level=logging.DEBUG,
    format='%(asctime)s - %(levelname)s - %(message)s'
)

class RedditStreamApp(App):
    """A terminal app for streaming Reddit comments."""
    
    TITLE = "Reddit Stream Console"
    BINDINGS = [
        ("q", "quit", "Quit"),
        ("r", "refresh", "Refresh"),
        ("escape", "show_menu", "Menu"),
        ("end", "scroll_bottom", "Scroll to Bottom"),
    ]
    
    CSS = """
    Screen {
        background: #1a1a1a;
    }

    .url-input-screen {
        background: #1a1a1a;
        align: center middle;
        width: 100%;
        height: 100%;
    }

    .url-input-label {
        color: #FFE6A9;
        text-align: center;
        text-style: bold;
        margin-bottom: 1;
    }

    .url-input-field {
        background: #2a2a2a;
        color: #FFE6A9;
        border: solid #659287;
        width: 90%;
        margin: 1 0;
        padding: 0 1;
    }

    .url-input-spacer {
        height: 1;
    }

    .url-input-buttons {
        width: 90%;
        height: auto;
        align: center middle;
        margin-top: 1;
    }

    .url-input-buttons Button {
        min-width: 20;
    }

    .url-submit-button, .url-cancel-button {
        width: 20;
        margin: 0 1;
        background: #659287;
        color: #FFE6A9;
        border: none;
    }

    .url-submit-button:hover, .url-cancel-button:hover {
        background: #DEAA79;
        color: #1a1a1a;
    }

    .url-submit-button:focus, .url-cancel-button:focus {
        background: #DEAA79;
        color: #1a1a1a;
        text-style: bold;
    }

    Header {
        background: #659287;
        color: #FFE6A9;
        text-style: bold;
    }

    Footer {
        background: #659287;
        color: #FFE6A9;
    }

    Footer > .footer--key {
        background: #659287;
        color: #DEAA79;
        text-style: bold;
    }

    Footer > .footer--description {
        background: #659287;
        color: #FFE6A9;
    }

    Button {
        margin: 1;
        min-width: 16;
        border: none;
        background: transparent;
        color: #DEAA79;
        text-align: center;
    }

    Button:hover, Button:focus {
        color: #FFE6A9;
        background: transparent;
    }

    .menu-screen, .thread-list {
        background: #1a1a1a;
        color: #B1C29E;
        align: center middle;
        padding: 1;
    }

    .menu-title, .thread-title {
        color: #FFE6A9;
        text-style: bold;
        text-align: center;
    }

    .menu-subtitle {
        color: #B1C29E;
        text-align: center;
        margin-bottom: 1;
    }

    .menu-button, .thread-button {
        width: 100%;
        text-align: center;
        margin: 0;
        padding: 0;
    }

    .back-button {
        margin-top: 1;
        color: #659287;
    }

    .back-button:hover, .back-button:focus {
        color: #FFE6A9;
    }

    CommentContainer {
        background: #1a1a1a;
        height: 1fr;
        border: solid #659287;
        overflow-y: scroll;
        display: none;  /* Hide by default */
        scrollbar-background: #1a1a1a;
        scrollbar-color: #659287;
        scrollbar-color-hover: #659287;
        scrollbar-color-active: #659287;
        scrollbar-background-active: #1a1a1a;
        scrollbar-background-hover: #1a1a1a;
        padding: 0 1;  /* Add horizontal padding */
    }

    CommentContainer.show {
        display: block;
    }

    CommentWidget {
        margin: 0;
        padding: 0;
        height: auto;
    }

    .comment-header {
        margin: 0;
        padding: 0 0 0 1;  /* Add left padding */
        height: 1;
    }

    .comment-body {
        margin: 0;
        padding: 0 0 1 1;  /* Add left padding */
        height: auto;
        color: #ffffff;
    }

    .thread-item {
        margin: 0;
        padding: 0;
        color: #B1C29E;
    }

    .thread-item:hover {
        color: #FFE6A9;
    }

    .thread-item-selected {
        color: #DEAA79;
        text-style: bold;
    }

    .notification {
        padding: 1 2;
        background: #659287;
        color: #FFE6A9;
    }

    .notification.error {
        background: #8B0000;
        color: #FFE6A9;
    }
    """
    
    def __init__(self):
        super().__init__()
        self.thread_finder = None
        self.current_thread = None
        self.showing_menu = True
        self.current_threads = []
        self.refresh_timer = None
        self.reddit_service = RedditService()
        self.menu_config = load_menu_config()
        self.menu_screen = None
        
    def compose(self) -> ComposeResult:
        """Create child widgets for the app."""
        yield Header()
        container = CommentContainer(id="comments-container")
        yield container
        self.menu_screen = MenuScreen()
        yield self.menu_screen
        yield Footer()
    
    def update_footer(self):
        """Update the footer with key bindings."""
        footer = self.query_one(Footer)
        keys_text = " ".join(f"[{key}] {action}" for key, action, _ in self.BINDINGS)
        footer.markup = keys_text
    
    def update_header(self, title: str = None):
        """Update the app title."""
        self.title = title if title else self.TITLE
    
    async def on_mount(self) -> None:
        """Start the refresh timer when app is mounted."""
        # Initialize ThreadFinder
        self.thread_finder = ThreadFinder()
        
        # Refresh every 5 seconds
        self.refresh_timer = self.set_interval(5, self.action_refresh)
        
        # Load menu config and create buttons
        await self.menu_screen.load_menu_items(self.menu_config)
        
        # Initialize footer
        self.update_footer()
    
    async def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button presses for menu items."""
        try:
            button_id = event.button.id
            if button_id and button_id.startswith("menu-"):
                thread_type = button_id[5:]  # Remove "menu-" prefix
                
                # Handle URL input
                if thread_type == "url_input":
                    # Hide menu screen
                    self.menu_screen.styles.display = "none"
                    self.showing_menu = False
                    
                    # Show URL input screen
                    url_input = UrlInputScreen()
                    await self.mount(url_input)
                    return
                menu_item = next((item for item in self.menu_config if item["type"] == thread_type), None)
                if menu_item:
                    # Hide menu screen
                    self.menu_screen.styles.display = "none"
                    self.showing_menu = False
                    
                    # Load and show threads
                    self.notify(f"Loading {menu_item['title']}...")
                    logging.debug(f"Finding threads for type: {thread_type}")
                    self.current_threads = await self.thread_finder.find_threads(thread_type)
                    
                    if self.current_threads:
                        # Remove any existing thread list screens first
                        try:
                            thread_lists = self.query(ThreadListScreen)
                            for thread_list in thread_lists:
                                thread_list.remove()
                        except NoMatches:
                            pass

                        # Show thread selection screen
                        thread_list = ThreadListScreen(menu_item["title"], self.current_threads)
                        logging.debug("Mounting thread list screen")
                        await self.mount(thread_list)
                    else:
                        self.notify(f"No threads found for {menu_item['title']}", severity="error")
                        self.menu_screen.styles.display = "block"
                        self.showing_menu = True
                        self.menu_screen.focus_first_button()

            elif button_id and button_id.startswith("thread-"):
                thread_id = button_id[7:]  # Remove "thread-" prefix
                thread = next((t for t in self.current_threads if t.id == thread_id), None)
                if thread:
                    # Remove any existing thread list screens
                    try:
                        thread_lists = self.query(ThreadListScreen)
                        for thread_list in thread_lists:
                            thread_list.remove()
                    except NoMatches:
                        pass
                    
                    # Show comments container and selected thread
                    comments_container = self.query_one("#comments-container")
                    comments_container.add_class("show")
                    self.current_thread = thread
                    self.update_header(thread.title)  # Update header with thread title
                    await self.refresh_comments()
            
            elif button_id == "back-to-menu":
                # Hide comments container if visible
                try:
                    comments_container = self.query_one("#comments-container")
                    comments_container.remove_class("show")
                except NoMatches:
                    pass
                    
                # Remove thread list screen if present
                try:
                    thread_list = self.query_one(ThreadListScreen)
                    thread_list.remove()
                except NoMatches:
                    pass
                
                # Remove any existing menu screens except the original
                menu_screens = self.query(MenuScreen)
                if len(menu_screens) > 1:
                    for screen in list(menu_screens)[1:]:  # Keep the first one
                        screen.remove()
                
                # Show menu and reset header
                self.menu_screen.styles.display = "block"
                self.showing_menu = True
                self.menu_screen.focus_first_button()
                self.update_header()  # Reset header to default title
        
        except Exception as e:
            logging.error(f"Unhandled error in button press handler: {str(e)}")
            self.notify("An unexpected error occurred", severity="error")
            # Ensure menu screen is visible in case of error
            try:
                menu_screen = self.query_one(MenuScreen)
                menu_screen.styles.display = "block"
                self.showing_menu = True
                self.menu_screen.focus_first_button()  # Restore focus when returning to menu
            except Exception as menu_error:
                logging.error(f"Failed to restore menu screen: {str(menu_error)}")
    
    async def action_show_menu(self) -> None:
        """Show the menu screen."""
        try:
            # Hide comments container if visible
            try:
                comments_container = self.query_one("#comments-container")
                comments_container.remove_class("show")
            except NoMatches:
                pass

            # Remove any existing thread list screens
            try:
                thread_lists = self.query(ThreadListScreen)
                for thread_list in thread_lists:
                    thread_list.remove()
            except NoMatches:
                pass

            # Reset current thread
            self.current_thread = None
            
            # Show menu and focus first button
            self.menu_screen.styles.display = "block"
            self.showing_menu = True
            self.menu_screen.focus_first_button()

        except Exception as e:
            logging.error(f"Error showing menu: {str(e)}")
            self.notify("An error occurred while showing the menu", severity="error")

    async def refresh_comments(self) -> None:
        """Fetch and display new comments."""
        if not self.current_thread:
            return
        
        try:
            # Fetch new comments
            comments = await self.reddit_service.fetch_comments(self.current_thread.permalink)
            
            if not comments:
                return
            
            # Update comments in container
            container = self.query_one(CommentContainer)
            await container.update_comments(comments)
            
        except Exception as e:
            logging.error(f"Error refreshing comments: {str(e)}")
            self.notify(f"Error refreshing comments: {str(e)}", severity="error")
    
    async def action_refresh(self) -> None:
        """Refresh comments when R is pressed or timer triggers."""
        if self.current_thread and not self.showing_menu:
            await self.refresh_comments()
    
    async def action_scroll_bottom(self) -> None:
        """Scroll to bottom when End is pressed."""
        container = self.query_one(CommentContainer)
        container.clear_user_scroll()
        await container.scroll_to_bottom()
    
    async def on_url_input_screen_url_submitted(self, message: UrlInputScreen.UrlSubmitted) -> None:
        """Handle URL submission from the URL input screen."""
        try:
            # Remove URL input screen
            url_input = self.query_one(UrlInputScreen)
            url_input.remove()
            
            # Load thread from URL
            self.notify("Loading thread...")
            thread = await self.thread_finder.get_thread_from_url(message.url)
            
            if thread:
                # Show comments container and selected thread
                comments_container = self.query_one("#comments-container")
                comments_container.add_class("show")
                self.current_thread = thread
                self.update_header(thread.title)
                await self.refresh_comments()
            else:
                self.notify("Invalid Reddit URL", severity="error")
                # Show menu again
                self.menu_screen.styles.display = "block"
                self.showing_menu = True
                self.menu_screen.focus_first_button()
                
        except Exception as e:
            logging.error(f"Error loading URL: {str(e)}")
            self.notify(f"Error loading URL: {str(e)}", severity="error")
            # Show menu again on error
            self.menu_screen.styles.display = "block"
            self.showing_menu = True
            self.menu_screen.focus_first_button()

    async def on_unmount(self) -> None:
        """Clean up when app is unmounted."""
        if self.refresh_timer:
            self.refresh_timer.stop()
        
        if self.thread_finder:
            await self.thread_finder.close()
