import aiohttp
import logging
from typing import List, Dict, Any
from datetime import datetime

class RedditService:
    """Service for interacting with Reddit's JSON API."""
    
    @staticmethod
    def _format_timestamp(timestamp: float) -> str:
        """Convert Unix timestamp to human readable format."""
        dt = datetime.fromtimestamp(timestamp)
        return dt.strftime("%Y-%m-%d %H:%M:%S")
    
    @staticmethod
    async def fetch_comments(thread_permalink: str) -> List[Dict[Any, Any]]:
        """Fetch comments for a thread using Reddit's JSON API."""
        comments_list = []
        
        # Remove leading/trailing slashes and add .json
        clean_permalink = thread_permalink.strip('/')
        url = f"https://www.reddit.com/{clean_permalink}.json"
        
        try:
            async with aiohttp.ClientSession() as session:
                logging.debug(f"Fetching comments from: {url}")
                async with session.get(url, headers={'User-Agent': 'RedditStreamApp/1.0'}) as response:
                    if response.status == 200:
                        data = await response.json()
                        
                        # Extract thread title and other metadata if needed
                        thread_data = data[0]['data']['children'][0]['data']
                        comments_data = data[1]['data']['children']
                        
                        # Get the post ID from the first item
                        post_id = thread_data['id']
                        
                        def process_comment(comment_data, depth=0):
                            """Process a single comment."""
                            if comment_data['kind'] != 't1':  # t1 = comment
                                return
                            
                            comment = comment_data['data']
                            
                            # Skip deleted/removed comments
                            if comment.get('body') in ['[deleted]', '[removed]']:
                                return
                            
                            # Format timestamp
                            created_utc = comment.get('created_utc', 0)
                            formatted_time = RedditService._format_timestamp(created_utc)
                            
                            # Get parent info
                            parent_id = comment.get('parent_id', '')
                            link_id = comment.get('link_id', '')
                            
                            # Only process if this is a direct reply to the post or to a known comment
                            parent_fullname = f"t3_{post_id}"  # Full name of the post
                            if parent_id == parent_fullname:
                                # This is a top-level comment
                                depth = 0
                            elif parent_id.startswith('t1_'):
                                # This is a reply to another comment
                                return None  # Skip for now, will be processed when we handle its parent
                            else:
                                return None
                            
                            comment_obj = {
                                "id": comment["id"],
                                "author": comment.get('author', '[deleted]'),
                                "body": comment.get('body', ''),
                                "created_utc": created_utc,
                                "formatted_time": formatted_time,
                                "score": comment.get('score', 0),
                                "depth": depth,
                                "parent_id": parent_id.split('_')[1]
                            }
                            
                            comments_list.append(comment_obj)
                            
                            # Now process replies to this comment
                            replies = comment.get('replies', '')
                            if isinstance(replies, dict) and 'data' in replies:
                                for reply in replies['data']['children']:
                                    if reply['kind'] == 't1':
                                        reply_data = reply['data']
                                        if reply_data.get('parent_id', '').split('_')[1] == comment["id"]:
                                            process_comment(reply, depth + 1)
                        
                        # First process top-level comments and their direct replies
                        for comment_data in comments_data:
                            process_comment(comment_data)
                        
                        # Sort by creation time
                        comments_list.sort(key=lambda x: x["created_utc"])
                        
                        return comments_list
                    else:
                        logging.error(f"Failed to fetch comments: HTTP {response.status}")
                        return []
                        
        except Exception as e:
            logging.error(f"Error fetching comments: {str(e)}")
            return []
