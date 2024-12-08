from dataclasses import dataclass

@dataclass
class Thread:
    """Model class for Reddit threads."""
    id: str
    title: str
    permalink: str
    type: str
