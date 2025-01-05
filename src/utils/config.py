import json
from typing import Dict, List, Any

def load_config(file_path: str) -> Dict[str, Any]:
    """Load configuration from JSON file."""
    with open(file_path, 'r') as f:
        return json.load(f)

def load_menu_config() -> List[Dict[str, Any]]:
    """Load menu configuration from JSON file."""
    config = load_config('config/menu_config.json')
    return config.get("menu_items", [])

def is_debug_logging_enabled() -> bool:
    """Check if debug logging is enabled in app config."""
    config = load_config('config/app_config.json')
    return config.get("debug_logging", False)
