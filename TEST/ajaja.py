# This is a simplified example and may not be fully functional. 
# Extracting age from YouTube is against their terms of service. 

import requests
from bs4 import BeautifulSoup
from googleapiclient.discovery import build

# Replace with your YouTube API key
api_key = "YOUR_API_KEY"

def search_videos(query, max_results=10):
  youtube = build("youtube", "v3", developerKey=api_key)
  request = youtube.search().list(
    q=query,
    part="id,snippet",
    type="video",
    maxResults=max_results
  )
  response = request.execute()
  return response['items']

def get_channel_id(video_id):
  youtube = build("youtube", "v3", developerKey=api_key)
  request = youtube.videos().list(
    part="snippet",
    id=video_id
  )
  response = request.execute()
  return response['items'][0]['snippet']['channelId']

def get_channel_details(channel_id):
  youtube = build("youtube", "v3", developerKey=api_key)
  request = youtube.channels().list(
    part="snippet",
    id=channel_id
  )
  response = request.execute()
  return response['items'][0]['snippet']

# Note: Directly determining age from YouTube data is not possible. 
# This script focuses on searching for videos based on keywords.

search_query = "20 year old creator"  # Replace with relevant keywords 
videos = search_videos(search_query)

for video in videos:
    video_id = video['id']['videoId']
    channel_id = get_channel_id(video_id)
    channel_details = get_channel_details(channel_id)

    # Print video and channel details
    print(f"Video Title: {video['snippet']['title']}")
    print(f"Channel: {channel_details['title']}")
    print(f"Video URL: https://www.youtube.com/watch?v={video_id}")
    print("-" * 30)

# Remember to comply with YouTube's API Terms of Service. 