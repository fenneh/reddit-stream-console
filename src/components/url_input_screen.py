from textual.app import ComposeResult
from textual.message import Message
from textual.containers import Vertical
from textual.widgets import Button, Label, Input
from textual.binding import Binding
from textual.events import Key

class UrlInputScreen(Vertical):
    """A screen for entering Reddit URLs."""

    BINDINGS = [
        Binding("escape", "cancel", "Cancel")
    ]

    def __init__(self):
        super().__init__()
        self.classes = "url-input-screen"

    def on_mount(self) -> None:
        """Focus the input field when the screen is mounted."""
        self.query_one("#url-input").focus()

    def compose(self) -> ComposeResult:
        """Create child widgets for the URL input screen."""
        yield Label("\nEnter Reddit URL", classes="url-input-label")
        yield Input(placeholder="https://reddit.com/r/...", id="url-input", classes="url-input-field")
        yield Label("", classes="url-input-spacer")  # Add spacing
        with Vertical(classes="url-input-buttons"):
            yield Button("Submit", id="submit-url", classes="url-submit-button")
            yield Button("Cancel", id="cancel-url", classes="url-cancel-button")

    def action_cancel(self) -> None:
        """Handle escape key to cancel."""
        self.remove()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button presses."""
        if event.button.id == "submit-url":
            self.submit_url()
        elif event.button.id == "cancel-url":
            self.remove()

    async def on_key(self, event: Key) -> None:
        """Handle key events."""
        if event.key == "enter":
            await self.submit_url()

    async def submit_url(self) -> None:
        """Submit the URL."""
        input_field = self.query_one("#url-input")
        if input_field.value:
            self.post_message(self.UrlSubmitted(input_field.value))

    class UrlSubmitted(Message):
        """Message sent when a URL is submitted."""
        def __init__(self, url: str):
            self.url = url
            super().__init__()
