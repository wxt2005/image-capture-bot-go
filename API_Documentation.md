# Image Capture Bot API Documentation

This document provides information about the API endpoints available in the Image Capture Bot service.

## Overview

The Image Capture Bot provides two main endpoints:
1. Telegram Webhook Handler (`/webhook`) - Processes incoming messages from Telegram
2. Direct API (`/api/send`) - Allows direct submission of URLs for processing

## Authentication

### Telegram Authentication

For Telegram users, authentication is required to use the bot's features:

1. Send the `/auth [key]` command to the bot, where `[key]` matches the configured `telegram.auth_key` value
2. Once authenticated, your user ID is stored in the database
3. To revoke authentication, use the `/revoke` command

### API Authentication

The direct API endpoint does not currently implement authentication. It is recommended to secure this endpoint at the network level.

## Endpoints

### 1. Telegram Webhook Handler

**Endpoint**: `/webhook` (or as configured)

**Method**: POST

**Description**: Processes incoming updates from the Telegram Bot API.

**Request Body**: Telegram Update object (as defined by the [Telegram Bot API](https://core.telegram.org/bots/api#update))

**Supported Commands**:
- `/start` - Sends a welcome message
- `/auth [key]` - Authenticates the user with the provided key
- `/revoke` - Revokes the user's authentication

**Callback Queries**:
- `like` - Adds a like to a message
- `force` - Forces processing of a message even if URLs are duplicates (requires authentication)

**Response**:
```json
{
  "media": [
    {
      "FileName": "string",
      "URL": "string",
      "Type": "string",
      "Source": "string",
      "Service": "string",
      "Author": "string",
      "AuthorURL": "string",
      "Title": "string",
      "Description": "string"
    }
  ],
  "message": "success"
}
```

### 2. Direct API

**Endpoint**: `/api/send`

**Method**: POST

**Description**: Processes a list of URLs directly without going through Telegram.

**Request Body**:
```json
{
  "url": ["string"],
  "force": boolean
}
```

- `url`: Array of URLs to process
- `force`: (Optional) If true, bypasses duplicate checking. Default is false.

**Response**:
```json
{
  "media": [
    {
      "FileName": "string",
      "URL": "string",
      "Type": "string",
      "Source": "string",
      "Service": "string",
      "Author": "string",
      "AuthorURL": "string",
      "Title": "string",
      "Description": "string"
    }
  ],
  "message": "success"
}
```

## Response Messages

The API returns the following message types in the response:

- `success`: The request was processed successfully
- `duplicate`: One or more URLs were identified as duplicates

## Media Object

The Media object contains the following fields:

| Field | Type | Description |
|-------|------|-------------|
| FileName | string | Name of the media file |
| URL | string | URL to the media file |
| Type | string | Type of media (photo, video, animation) |
| Source | string | Source URL of the media |
| Service | string | Service that provided the media (Twitter, Tumblr, Pixiv, etc.) |
| Author | string | Author/creator of the media |
| AuthorURL | string | URL to the author's profile |
| Title | string | Title of the media |
| Description | string | Description of the media |

## Supported Services

The bot can extract media from the following services:

- Twitter
- Tumblr
- Pixiv
- Danbooru
- Misskey
- Bluesky
- Instagram

Media can be consumed by:

- Telegram
- S3

## Error Handling

The API uses standard HTTP status codes:

- `200 OK`: Request was successful
- `500 Internal Server Error`: An error occurred while processing the request

When an error occurs, the response body may not be returned.

## Duplicate Handling

By default, the API checks for duplicate URLs to avoid processing the same content multiple times. This behavior can be bypassed:

- In the Telegram interface: Using the "Force" button on a message
- In the direct API: Setting the `force` parameter to `true`

## Examples

### Example 1: Submitting URLs via the API

**Request**:
```http
POST /api/send
Content-Type: application/json

{
  "url": [
    "https://twitter.com/user/status/123456789",
    "https://www.pixiv.net/en/artworks/12345678"
  ],
  "force": false
}
```

**Response**:
```json
{
  "media": [
    {
      "FileName": "image1.jpg",
      "URL": "https://example.com/image1.jpg",
      "Type": "photo",
      "Source": "https://twitter.com/user/status/123456789",
      "Service": "Twitter",
      "Author": "TwitterUser",
      "AuthorURL": "https://twitter.com/user",
      "Title": "",
      "Description": "Tweet content"
    },
    {
      "FileName": "image2.jpg",
      "URL": "https://example.com/image2.jpg",
      "Type": "photo",
      "Source": "https://www.pixiv.net/en/artworks/12345678",
      "Service": "Pixiv",
      "Author": "PixivUser",
      "AuthorURL": "https://www.pixiv.net/users/12345",
      "Title": "Artwork Title",
      "Description": "Artwork description"
    }
  ],
  "message": "success"
}
```

### Example 2: Duplicate URL Response

**Request**:
```http
POST /api/send
Content-Type: application/json

{
  "url": [
    "https://twitter.com/user/status/123456789"
  ],
  "force": false
}
```

**Response** (if the URL was previously processed):
```json
{
  "media": null,
  "message": "duplicate"
}
```
