import json
from typing import Dict, List, Any

def load_menu_config() -> List[Dict[str, Any]]:
    """Load menu configuration from JSON file."""
    with open('config/menu_config.json', 'r') as f:
        config = json.load(f)
        return config.get("menu_items", [])
