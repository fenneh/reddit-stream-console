from textual.app import ComposeResult
from textual.containers import Vertical
from textual.widgets import Button, Label
from textual.binding import Binding

class MenuScreen(Vertical):
    """A screen that displays menu options."""

    BINDINGS = [
        Binding("up", "focus_previous", "Previous", show=False),
        Binding("down", "focus_next", "Next", show=False),
        Binding("enter", "select_focused", "Select", show=False)
    ]

    def __init__(self):
        super().__init__()
        self.buttons = []
        self.current_focus = -1

    async def load_menu_items(self, menu_config):
        """Load menu items from config."""
        self.buttons = []  # Clear existing buttons
        for item in menu_config:
            if "title" in item:  # Only create button if item has a title
                button = Button(item["title"], id=f"menu-{item['type']}", classes="menu-button")
                self.buttons.append(button)
                await self.mount(button)
        
        # Focus first button if any exist
        if self.buttons:
            self.current_focus = 0
            self.focus_first_button()

    def focus_first_button(self) -> None:
        """Focus the first button in the list."""
        if self.buttons:
            self.current_focus = 0
            self.buttons[0].focus()

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
