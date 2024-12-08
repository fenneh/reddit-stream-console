from textual.app import ComposeResult
from textual.containers import ScrollableContainer
from textual.widgets import Static, Label
from datetime import datetime, timezone
import logging
from typing import Dict, Set

class CommentWidget(Static):
    """A widget to display a single comment."""

    def __init__(self, author: str, body: str, timestamp: str, score: int, depth: int = 0, comment_id: str = ""):
        super().__init__()
        self.author = author
        self.body = body
        self.timestamp = timestamp
        self.score = score
        self.depth = depth
        self.comment_id = comment_id
        
    def compose(self) -> ComposeResult:
        """Create child widgets for the comment."""
        indent = "  " * self.depth
        arrow = "[#DEAA79]→[/] " if self.depth > 0 else ""
        
        header = f"{indent}{arrow}[#FFE6A9]{self.author}[/] • [#B1C29E]{self.score} points[/] • [#659287]{self.timestamp}[/]"
        yield Label(header, classes="comment-header")
        yield Label(f"{indent}{self.body}", classes="comment-body")

class CommentContainer(ScrollableContainer):
    """A container for comments that supports auto-scrolling."""

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.user_scrolled = False
        self.comment_ids = set()  # Track existing comment IDs
        self.comment_widgets = {}  # Map comment IDs to widgets
    
    async def _on_scroll_up(self) -> None:
        """Handle scroll up event."""
        await super()._on_scroll_up()
        self.user_scrolled = True
    
    async def _on_scroll_down(self) -> None:
        """Handle scroll down event."""
        await super()._on_scroll_down()
        # If we're at the bottom, allow auto-scroll again
        if self.scroll_offset.y >= self.content_size.height - self.size.height:
            self.user_scrolled = False
    
    def clear_user_scroll(self):
        """Reset the user scroll state."""
        self.user_scrolled = False
    
    async def update_comments(self, comments):
        """Update comments, only adding new ones and updating existing ones."""
        try:
            current_scroll = self.scroll_offset.y
            was_at_bottom = (current_scroll >= self.content_size.height - self.size.height)
            
            # Process new comments
            new_comments = []
            for comment in comments:
                comment_id = comment["id"]
                if comment_id not in self.comment_ids:
                    widget = CommentWidget(
                        author=comment["author"],
                        body=comment["body"],
                        timestamp=comment["formatted_time"],
                        score=comment["score"],
                        depth=comment["depth"],
                        comment_id=comment_id
                    )
                    new_comments.append(widget)
                    self.comment_ids.add(comment_id)
                    self.comment_widgets[comment_id] = widget
            
            # Add new comments if any
            if new_comments:
                for widget in new_comments:
                    await self.mount(widget)
                
                # If we were at the bottom before adding new comments,
                # or haven't manually scrolled, scroll to bottom
                if was_at_bottom or not self.user_scrolled:
                    await self.scroll_end(animate=False)
                else:
                    # Maintain scroll position
                    await self.scroll_to(y=current_scroll, animate=False)
        
        except Exception as e:
            logging.error(f"Error updating comments: {str(e)}")
    
    async def scroll_to_bottom(self):
        """Scroll to the bottom if user hasn't manually scrolled up."""
        if not self.user_scrolled:
            try:
                await self.scroll_end(animate=False)
            except Exception as e:
                logging.error(f"Error scrolling to bottom: {str(e)}")
