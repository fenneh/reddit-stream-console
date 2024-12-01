import aiohttp
import asyncio
import os
from datetime import datetime
import asyncpraw
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

async def get_thread_info():
    # Initialize Reddit instance
    reddit = asyncpraw.Reddit(
        client_id=os.getenv('REDDIT_CLIENT_ID'),
        client_secret=os.getenv('REDDIT_CLIENT_SECRET'),
        user_agent="console:reddit-stream-debug:v1.0 (by /u/your_username)"
    )

    try:
        submission_id = "1h2op8h"
        submission = await reddit.submission(id=submission_id)
        print(f"\nThread Info:")
        print(f"Title: {submission.title}")
        print(f"Created: {datetime.utcfromtimestamp(submission.created_utc).strftime('%Y-%m-%d %H:%M:%S UTC')}")
        print(f"Author: {submission.author}")
        print(f"URL: {submission.url}")
    except Exception as e:
        print(f"Error getting thread info: {str(e)}")
    finally:
        await reddit.close()

async def get_newest_comments():
    headers = {
        "accept": "*/*",
        "accept-language": "en-GB,en;q=0.9,en-US;q=0.8",
        "user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
    }
    
    submission_id = "1h2op8h"
    url = f"https://www.reddit.com/comments/{submission_id}.json?sort=new&limit=20&raw_json=1"
    
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(url, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    
                    # Comments are in the second array
                    comments = data[1]['data']['children']
                    
                    print(f"\nFound {len(comments)} newest comments")
                    print("-" * 80)
                    
                    for i, comment in enumerate(comments, 1):
                        comment_data = comment['data']
                        created_time = datetime.utcfromtimestamp(comment_data['created_utc']).strftime('%Y-%m-%d %H:%M:%S UTC')
                        print(f"\n{i}. Time: {created_time}")
                        print(f"Author: {comment_data['author']}")
                        print(f"Content: {comment_data['body'][:100]}...")
                else:
                    print(f"Error: Got status code {response.status}")
                    
    except Exception as e:
        print(f"Error getting comments: {str(e)}")

async def main():
    await get_thread_info()
    await get_newest_comments()

if __name__ == "__main__":
    current_time = datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S UTC')
    print(f"Current time: {current_time}")
    
    asyncio.run(main())
