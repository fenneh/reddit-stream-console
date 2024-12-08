from textual.app import ComposeResult
from textual.containers import Center, Vertical
from textual.widgets import Static, Label, Button
from textual.binding import Binding

class ThreadListScreen(Vertical):
    """Screen showing available threads for a selected type."""
    
    BINDINGS = [
        Binding("up", "focus_previous", "Previous", show=False),
        Binding("down", "focus_next", "Next", show=False),
        Binding("enter", "select_focused", "Select", show=False)
    ]

    def __init__(self, thread_type: str, threads: list):
        super().__init__()
        self.thread_type = thread_type
        self.threads = threads
        self.buttons = []
        self.current_focus = -1
    
    def compose(self) -> ComposeResult:
        """Create child widgets for the thread list."""
        with Center():
            with Vertical(classes="thread-list-container"):
                yield Label("Select a thread to view:", classes="subtitle")
                yield Static("", classes="spacer")
                
                for thread in self.threads:
                    button = Button(
                        thread.title,
                        id=f"thread-{thread.id}",
                        classes="thread-button"
                    )
                    self.buttons.append(button)
                    yield button
                
                yield Static("", classes="spacer")
                yield Button("← Back to Menu", id="back-to-menu", classes="back-button")
                
                # Focus first button if this is the first one
                if len(self.buttons) == len(self.threads):
                    self.current_focus = 0
                    self.buttons[self.current_focus].focus()

    def action_focus_next(self) -> None:
        """Focus the next button in the list."""
        if not self.buttons:
            return
        
        self.current_focus = (self.current_focus + 1) % len(self.buttons)
        self.buttons[self.current_focus].focus()

    def action_focus_previous(self) -> None:
        """Focus the previous button in the list."""
        if not self.buttons:
            return
        
        self.current_focus = (self.current_focus - 1) % len(self.buttons)
        self.buttons[self.current_focus].focus()

    def action_select_focused(self) -> None:
        """Simulate click on the focused button."""
        if 0 <= self.current_focus < len(self.buttons):
            self.buttons[self.current_focus].press()
