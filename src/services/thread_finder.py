import asyncpraw
import os
import logging
from typing import List
from datetime import datetime, timezone, timedelta
from ..models.thread import Thread
from ..utils.config import load_menu_config

class ThreadFinder:
    """Class to find relevant Reddit threads."""
    
    def __init__(self):
        self.reddit = asyncpraw.Reddit(
            client_id=os.getenv('REDDIT_CLIENT_ID'),
            client_secret=os.getenv('REDDIT_CLIENT_SECRET'),
            user_agent=os.getenv('REDDIT_USER_AGENT', 'RedditStreamApp 1.0')
        )
        self.menu_config = load_menu_config()

    async def find_threads(self, thread_type: str) -> List[Thread]:
        """Find threads based on the specified type."""
        try:
            # Get config for thread type
            thread_config = next(
                (item for item in self.menu_config if item["type"] == thread_type),
                None
            )
            
            if not thread_config:
                logging.error(f"No configuration found for thread type: {thread_type}")
                return []

            logging.debug(f"Finding threads with config: {thread_config}")
            
            # Get subreddit
            subreddit = await self.reddit.subreddit(thread_config["subreddit"])
            
            # Calculate age limit
            max_age = thread_config.get("max_age_hours", 24)
            age_limit = datetime.now(timezone.utc) - timedelta(hours=max_age)
            
            # Build search query
            flair_list = [thread_config["flair"]] if isinstance(thread_config["flair"], str) else thread_config["flair"]
            # Try each flair in the list until we find matches
            threads = []
            for flair in flair_list:
                # Reddit search requires exact flair match with quotes
                flair_query = f'flair:"{flair}"'
                logging.debug(f"Using search with query: {flair_query}")
                async_generator = subreddit.search(
                    query=flair_query,
                    sort='new',
                    time_filter='week',  # Changed from day to week for more results
                    limit=thread_config.get("limit", 50)
                )
                
                async for submission in async_generator:
                    try:
                        # Skip if too old
                        created_utc = datetime.fromtimestamp(submission.created_utc, tz=timezone.utc)
                        if created_utc < age_limit:
                            logging.debug(f"Skipping old thread: {submission.title} (created {created_utc})")
                            continue
                        
                        # Check title must contain
                        if "title_must_contain" in thread_config:
                            if not any(phrase.lower() in submission.title.lower() 
                                    for phrase in thread_config["title_must_contain"]):
                                logging.debug(f"Title must contain mismatch for thread: {submission.title}")
                                continue
                        
                        # Check title must not contain
                        if "title_must_not_contain" in thread_config:
                            if any(phrase.lower() in submission.title.lower() 
                                for phrase in thread_config["title_must_not_contain"]):
                                logging.debug(f"Title must not contain match for thread: {submission.title}")
                                continue
                        
                        # Add matching thread
                        logging.debug(f"Found matching thread: {submission.title}")
                        threads.append(Thread(
                            id=submission.id,
                            title=submission.title,
                            permalink=submission.permalink,
                            type=thread_type
                        ))
                        
                    except Exception as e:
                        logging.error(f"Error processing submission {submission.id}: {str(e)}")
                        continue
                
                # If we found threads with this flair, no need to try others
                if threads:
                    break
            
            logging.info(f"Found {len(threads)} matching threads for type {thread_type}")
            return threads

        except Exception as e:
            logging.error(f"Error finding threads: {str(e)}")
            return []

    async def get_thread_from_url(self, url: str) -> Thread:
        """Get a thread from a Reddit URL."""
        try:
            # Clean up URL to get permalink
            # Handle both full URLs and relative permalinks
            if url.startswith('http'):
                # Handle various Reddit domains
                for domain in ['reddit.com', 'old.reddit.com', 'sh.reddit.com']:
                    if domain in url:
                        permalink = url.split(domain)[-1]
                        break
                else:
                    permalink = url
            else:
                permalink = url
            
            # Remove trailing slash and .json if present
            permalink = permalink.rstrip('/').rstrip('.json')
            
            # Remove any query parameters
            if '?' in permalink:
                permalink = permalink.split('?')[0]
            
            # Extract thread ID from permalink
            parts = [p for p in permalink.split('/') if p]  # Split and remove empty parts
            if len(parts) >= 4 and parts[0] == 'r':
                thread_id = parts[3]  # Get ID from /r/subreddit/comments/ID/...
            else:
                raise ValueError("Invalid Reddit URL format")
            
            # Fetch submission to get title
            submission = await self.reddit.submission(id=thread_id)
            
            return Thread(
                id=thread_id,
                title=submission.title,
                permalink=permalink,
                type='url_input'
            )
            
        except Exception as e:
            logging.error(f"Error getting thread from URL: {str(e)}")
            return None

    async def close(self):
        """Close the Reddit client session."""
        await self.reddit.close()
