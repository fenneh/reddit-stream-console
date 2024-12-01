import asyncpraw
import os
from datetime import datetime, timedelta
import re
from typing import List, Dict, Optional
import time
import json
from pathlib import Path

class ThreadType:
    SOCCER_MATCH = "soccer_match"
    SOCCER_POST_MATCH = "soccer_post_match"
    NFL_GAME = "nfl_game"
    FPL_RANT = "fpl_rant"

    @staticmethod
    def load_config():
        """Load menu configuration from JSON file"""
        config_path = Path(__file__).parent / "config" / "menu_config.json"
        try:
            with open(config_path, 'r') as f:
                return json.load(f)
        except Exception as e:
            print(f"Error loading menu config: {str(e)}")
            return {"menu_items": []}

    @staticmethod
    def get_thread_config(thread_type: str) -> dict:
        """Get configuration for a specific thread type"""
        config = ThreadType.load_config()
        for item in config.get("menu_items", []):
            if item.get("type") == thread_type:
                return item
        return {}

class Thread:
    def __init__(self, title: str, url: str, created_utc: float, thread_type: str, score: int):
        self.title = title
        self.url = url
        self.created_utc = created_utc
        self.thread_type = thread_type
        self.score = score
        self.created_time = datetime.fromtimestamp(created_utc)

    def __str__(self):
        return f"[{self.thread_type}] {self.title} ({self.created_time.strftime('%Y-%m-%d %H:%M:%S')})"

class ThreadFinder:
    def __init__(self):
        self.reddit = asyncpraw.Reddit(
            client_id=os.getenv('REDDIT_CLIENT_ID'),
            client_secret=os.getenv('REDDIT_CLIENT_SECRET'),
            user_agent="console:reddit-stream:v1.0 (by /u/your_username)"
        )

    async def find_threads(self, thread_type: str) -> List[Thread]:
        """Find threads based on configuration"""
        config = ThreadType.get_thread_config(thread_type)
        if not config:
            print(f"No configuration found for thread type: {thread_type}")
            return []

        threads = []
        seen_urls = set()
        cutoff_time = datetime.now() - timedelta(hours=config.get("max_age_hours", 12))

        try:
            print(f"Searching subreddit: {config['subreddit']} for flair: {config['flair']}")
            subreddit = await self.reddit.subreddit(config["subreddit"].lower())  # Convert to lowercase
            
            # Search for posts with specified flair
            try:
                search_query = f'flair:"{config["flair"]}"'
                time_window = 'week' if config.get("max_age_hours", 12) > 24 else 'day'
                print(f"Search query: {search_query}, Time window: {time_window}")
                
                async for submission in subreddit.search(search_query, sort='new', time_filter=time_window, limit=config.get("limit", 50)):
                    created_time = submission.created_utc
                    created_dt = datetime.fromtimestamp(created_time)
                    
                    if created_dt >= cutoff_time:
                        # Check title filters if they exist
                        title = submission.title.lower()
                        
                        # Check for required title content
                        must_contain = config.get("title_must_contain", [])
                        if must_contain and not any(phrase.lower() in title for phrase in must_contain):
                            continue
                            
                        # Check for forbidden title content
                        must_not_contain = config.get("title_must_not_contain", [])
                        if must_not_contain and any(phrase.lower() in title for phrase in must_not_contain):
                            continue
                        
                        url = f"https://www.reddit.com{submission.permalink}"
                        if url not in seen_urls:
                            seen_urls.add(url)
                            threads.append(Thread(
                                title=submission.title,
                                url=url,
                                created_utc=created_time,
                                thread_type=thread_type,
                                score=submission.score
                            ))
            except Exception as e:
                print(f"Error searching threads by flair: {str(e)}")
                print(f"Subreddit: {config['subreddit']}, Thread type: {thread_type}")
                return []

            threads_found = sorted(threads, key=lambda x: x.created_utc, reverse=True)
            print(f"Found {len(threads_found)} threads for {thread_type}")
            return threads_found
            
        except Exception as e:
            print(f"Error in thread finder: {str(e)}")
            print(f"Failed to search subreddit: {config.get('subreddit', 'unknown')}")
            return []

    async def close(self):
        await self.reddit.close()
