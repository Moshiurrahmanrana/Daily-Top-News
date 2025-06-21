# Top News API

A simple Go REST API to fetch top daily news headlines, images, and descriptions from popular news websites, including The Daily Star (Bangladesh), BBC, CNN, Reuters, and TechCrunch.

---

## 🚀 Features
- Get top news from multiple sources in one place
- Easy-to-use HTTP API
- Returns headlines, images, descriptions, and links
- **Automatic image scraping** - Fetches missing images from article pages
- Beginner-friendly setup

---

## 📰 Supported News Sources
- **The Daily Star** (Bangladesh) - *Currently Active*
- **BBC News**
- **CNN**
- **Reuters**
- **TechCrunch**

---

## 🖼️ Image Scraping Feature

The API now includes intelligent image scraping functionality:

- **Automatic Detection**: When articles don't have images from the main page, the API automatically visits the article URL to extract images
- **Multiple Fallback Methods**: Uses various techniques to find images:
  - `<picture>` tags with `data-srcset`
  - Gallery spans with `data-src`
  - Open Graph meta tags (`og:image`)
  - Article body images
- **Rate Limiting**: Includes delays between requests to avoid overwhelming servers
- **Error Handling**: Gracefully handles cases where images cannot be found

---

## 🛠️ Setup Instructions

### 1. Prerequisites
- [Go](https://go.dev/dl/) 1.21 or newer
- (Optional) [Docker](https://www.docker.com/) if you want to use containers

### 2. Download & Install

Clone this repository and enter the folder:
```bash
git clone <your-repo-url>
cd Top-News
```

Download dependencies:
```bash
go mod tidy
```

### 3. Run the API Server
```bash
go run .
```

The server will start at: [http://localhost:8080](http://localhost:8080)

---

## 🧑‍💻 API Usage

### Get all news from all sources
```
GET /api/v1/news
```

### Get news from a specific source
```
GET /api/v1/news/{source}
```
Replace `{source}` with one of:
- `thedailystar`
- `bbc`
- `cnn`
- `reuters`
- `techcrunch`

**Example:**
```
GET /api/v1/news/thedailystar
```

### List all available sources
```
GET /api/v1/sources
```

### Health check
```
GET /api/v1/health
```

---

## 📦 Example Response

```
{
  "success": true,
  "data": [
    {
      "id": "thedailystar_0",
      "title": "Headline here",
      "description": "Short description here",
      "image_url": "https://...jpg",
      "url": "https://...",
      "source": "thedailystar",
      "published_at": "2025-06-21T13:00:00Z"
    }
  ],
  "count": 1,
  "source": "thedailystar"
}
```

---

## 🐳 Docker (Optional)

Build and run with Docker:
```bash
docker build -t top-news-api .
docker run -p 8080:8080 top-news-api
```

---

## 📝 Notes
- This API scrapes public news websites. If a site changes its layout, results may break.
- **Image scraping** may take additional time for articles without images on the main page.
- For production, consider using official news APIs or RSS feeds for stability.
- Please respect the terms of service of each news source.

---

## 🤝 Contributing
Pull requests are welcome! For major changes, please open an issue first.

---

## 📄 License
MIT # Daily-Top-News
