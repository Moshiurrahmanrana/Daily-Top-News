package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"top-news/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly/v2"
)

// NewsService handles news fetching operations
type NewsService struct {
	sources map[string]models.Source
	client  *http.Client
}

// NewNewsService creates a new news service instance
func NewNewsService() *NewsService {
	// Initialize news sources - only The Daily Star and CNN
	sources := map[string]models.Source{
		"thedailystar": {
			Name:        "thedailystar",
			DisplayName: "The Daily Star",
			URL:         "https://www.thedailystar.net/",
			Active:      true,
		},
		"cnn": {
			Name:        "cnn",
			DisplayName: "CNN",
			URL:         "https://edition.cnn.com/",
			Active:      true,
		},
	}

	// Create HTTP client with timeout and redirect handling
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Allow all redirects
		},
	}

	return &NewsService{
		sources: sources,
		client:  client,
	}
}

// GetAllNews fetches news from all active sources
func (ns *NewsService) GetAllNews(c *gin.Context) {
	var wg sync.WaitGroup
	allNews := make(chan []models.NewsArticle, len(ns.sources))
	
	// Fetch news from all sources concurrently
	for name, source := range ns.sources {
		if !source.Active {
			continue
		}
		
		wg.Add(1)
		go func(sourceName string, source models.Source) {
			defer wg.Done()
			news, err := ns.fetchNewsFromSource(sourceName, source.URL)
			if err != nil {
				log.Printf("Error fetching from %s: %v", sourceName, err)
				allNews <- []models.NewsArticle{}
				return
			}
			allNews <- news
		}(name, source)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(allNews)
	}()

	// Collect all news
	var allArticles []models.NewsArticle
	for news := range allNews {
		allArticles = append(allArticles, news...)
	}

	response := models.NewsResponse{
		Success: true,
		Data:    allArticles,
		Count:   len(allArticles),
	}

	c.JSON(http.StatusOK, response)
}

// GetNewsBySource fetches news from a specific source
func (ns *NewsService) GetNewsBySource(c *gin.Context) {
	sourceName := c.Param("source")
	
	source, exists := ns.sources[sourceName]
	if !exists {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   "source_not_found",
			Message: "News source not found",
		})
		return
	}

	if !source.Active {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "source_inactive",
			Message: "News source is currently inactive",
		})
		return
	}

	news, err := ns.fetchNewsFromSource(sourceName, source.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "fetch_error",
			Message: fmt.Sprintf("Failed to fetch news: %v", err),
		})
		return
	}

	response := models.NewsResponse{
		Success: true,
		Data:    news,
		Count:   len(news),
		Source:  sourceName,
	}

	c.JSON(http.StatusOK, response)
}

// GetAvailableSources returns all available news sources
func (ns *NewsService) GetAvailableSources(c *gin.Context) {
	var sources []models.Source
	for _, source := range ns.sources {
		sources = append(sources, source)
	}

	response := models.SourcesResponse{
		Success: true,
		Sources: sources,
	}

	c.JSON(http.StatusOK, response)
}

// fetchNewsFromSource fetches news from a specific source
func (ns *NewsService) fetchNewsFromSource(sourceName, url string) ([]models.NewsArticle, error) {
	// Only handle The Daily Star
	if sourceName == "thedailystar" {
		return ns.fetchTheDailyStarWithColly(url)
	}
	if sourceName == "cnn" {
		return ns.fetchCNNWithColly(url)
	}

	return nil, fmt.Errorf("unsupported source: %s", sourceName)
}

// fetchTheDailyStarWithColly fetches news from The Daily Star using Colly
func (ns *NewsService) fetchTheDailyStarWithColly(url string) ([]models.NewsArticle, error) {
	// Initialize a slice to store articles
	articles := []models.NewsArticle{}

	// Create a new Colly collector
	c := colly.NewCollector(
		colly.AllowedDomains("www.thedailystar.net", "thedailystar.net"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Add rate limiting to avoid server blocks
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*.thedailystar.net",
		Delay:       2 * time.Second,
		RandomDelay: 1 * time.Second,
	})

	// Counter for article IDs
	articleID := 0

	// OnHTML callback for article containers
	c.OnHTML(".story, .article, .news-item, .card, .pane-content, .teaser, .post, .news-block", func(e *colly.HTMLElement) {
		if len(articles) >= 10 { // Limit to 10 articles for testing
			return
		}

		// Extract title - get only the first/main title
		var title string
		titleSelectors := []string{"h1", "h2", "h3", "h4", ".title", ".headline"}
		for _, selector := range titleSelectors {
			title = strings.TrimSpace(e.ChildText(selector))
			if title != "" && len(title) >= 10 {
				break
			}
		}
		
		if title == "" {
			return
		}

		// Clean up title - remove extra whitespace and newlines
		title = strings.ReplaceAll(title, "\n", " ")
		title = strings.ReplaceAll(title, "\r", " ")
		title = strings.ReplaceAll(title, "\t", " ")
		title = strings.Join(strings.Fields(title), " ") // Normalize whitespace
		
		// Truncate at first comma or period to get only the main headline
		if idx := strings.Index(title, ","); idx != -1 {
			title = strings.TrimSpace(title[:idx])
		}
		if idx := strings.Index(title, "."); idx != -1 {
			title = strings.TrimSpace(title[:idx])
		}

		// Extract link
		link := e.ChildAttr("a", "href")
		if link == "" {
			return
		}
		link = e.Request.AbsoluteURL(link)

		// Filter out category links (e.g., /news/bangladesh)
		pathSegments := strings.Split(strings.TrimPrefix(link, "https://www.thedailystar.net"), "/")
		if len(pathSegments) <= 3 || pathSegments[len(pathSegments)-1] == "" {
			return
		}

		// Skip if not a news article link
		if !strings.Contains(link, "/news/") &&
			!strings.Contains(link, "/bangladesh/") &&
			!strings.Contains(link, "/world/") &&
			!strings.Contains(link, "/business/") &&
			!strings.Contains(link, "/sports/") &&
			!strings.Contains(link, "/entertainment/") {
			return
		}

		// Skip duplicates
		for _, article := range articles {
			if article.URL == link {
				return
			}
		}

		// Extract image URL
		imageURL := ""
		for _, attr := range []string{"src", "data-src", "data-lazy-src", "data-srcset", "data-original", "data-image", "data-lazy"} {
			imageURL = e.ChildAttr("img", attr)
			if imageURL != "" {
				break
			}
		}
		if imageURL == "" {
			imageURL = e.ChildAttr("picture source", "srcset")
		}
		if imageURL != "" {
			imageURL = e.Request.AbsoluteURL(imageURL)
			if strings.Contains(imageURL, ",") {
				imageURL = strings.Split(imageURL, ",")[0]
				imageURL = strings.TrimSpace(strings.Split(imageURL, " ")[0])
			}
		}

		// Extract description
		description := strings.TrimSpace(e.ChildText("p, .summary, .intro, .teaser-text, .excerpt, .description"))
		if description != "" && len(description) > 200 {
			description = description[:200] + "..."
		}

		// Create NewsArticle struct
		article := models.NewsArticle{
			ID:          fmt.Sprintf("dailystar_%d", articleID),
			Title:       title,
			Description: description,
			ImageURL:    imageURL,
			URL:         link,
			Source:      "thedailystar",
			PublishedAt: time.Now(),
		}

		articles = append(articles, article)
		articleID++
	})

	// OnError callback to handle errors
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error: %v, Status Code: %d", err, r.StatusCode)
	})

	// Start scraping the homepage
	err := c.Visit(url)
	if err != nil {
		return nil, fmt.Errorf("failed to visit: %v", err)
	}

	// Wait for all requests to complete
	c.Wait()

	// Update missing image URLs by scraping individual article pages
	//ns.updateMissingImageURLs(&articles)
	//ns.updateMissingImageURLs(&articles)
	ns.updateArticleDetails(&articles)

	return articles, nil
}

// fetchCNNWithColly fetches news from CNN using Colly
func (ns *NewsService) fetchCNNWithColly(url string) ([]models.NewsArticle, error) {
	// Initialize a slice to store articles
	articles := []models.NewsArticle{}

	// Create a new Colly collector
	c := colly.NewCollector(
		colly.AllowedDomains("edition.cnn.com", "cnn.com"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Add rate limiting
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*.cnn.com",
		Delay:       2 * time.Second,
		RandomDelay: 1 * time.Second,
	})

	// Counter for article IDs
	articleID := 0

	// OnHTML callback for article containers
	c.OnHTML("a[data-link-type='article']", func(e *colly.HTMLElement) {
		if len(articles) >= 15 { // Limit to 15 articles for CNN
			return
		}

		link := e.Request.AbsoluteURL(e.Attr("href"))

		// Skip duplicates by URL
		for _, article := range articles {
			if article.URL == link {
				return
			}
		}

		var title string
		// CNN uses spans with data-editable="headline" for many titles
		title = e.ChildText("span[data-editable='headline']")
		if title == "" {
			// Fallback for different card styles
			title = e.ChildText(".container__headline-text")
		}
		title = strings.TrimSpace(title)

		if title == "" || len(title) < 10 {
			return
		}

		// Skip duplicates by Title
		for _, article := range articles {
			if article.Title == title {
				return
			}
		}

		article := models.NewsArticle{
			ID:          fmt.Sprintf("cnn_%d", articleID),
			Title:       title,
			Description: "", // Description is not easily available on the homepage
			ImageURL:    "", // Will be fetched by updateMissingImageURLs
			URL:         link,
			Source:      "cnn",
			PublishedAt: time.Now(),
		}

		articles = append(articles, article)
		articleID++
	})

	// OnError callback to handle errors
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error scraping CNN: %v, Status Code: %d", err, r.StatusCode)
	})

	// Start scraping the homepage
	err := c.Visit(url)
	if err != nil {
		return nil, fmt.Errorf("failed to visit CNN: %v", err)
	}

	// Wait for all requests to complete
	c.Wait()

	// Update missing image URLs by scraping individual article pages
	ns.updateArticleDetails(&articles)

	return articles, nil
}

// updateArticleDetails updates empty image_url and description fields by scraping from the article URL
func (ns *NewsService) updateArticleDetails(articles *[]models.NewsArticle) {
	for i := range *articles {
		article := &(*articles)[i]
		if article.ImageURL == "" || article.Description == "" {
			imageURL, description, err := ns.scrapeArticleDetailsFromURL(article.URL)
			if err != nil {
				log.Printf("Error scraping details for %s: %v", article.URL, err)
				continue
			}
			if article.ImageURL == "" && imageURL != "" {
				article.ImageURL = imageURL
			}
			if article.Description == "" && description != "" {
				article.Description = description
			}
			// Add delay to avoid overwhelming the server
			time.Sleep(1 * time.Second)
		}
	}
}

// scrapeArticleDetailsFromURL fetches an image URL and description from the given webpage
func (ns *NewsService) scrapeArticleDetailsFromURL(url string) (string, string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set User-Agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch URL %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse HTML using goquery
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML: %v", err)
	}

	// --- Scrape Image URL ---
	imageURL := ""
	doc.Find("picture img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("data-srcset"); exists && imageURL == "" {
			imageURL = src
		}
	})
	if imageURL == "" {
		doc.Find("span.lg-gallery").Each(func(i int, s *goquery.Selection) {
			if src, exists := s.Attr("data-src"); exists && imageURL == "" {
				imageURL = src
			}
		})
	}
	if imageURL == "" {
		doc.Find("meta[property='og:image']").Each(func(i int, s *goquery.Selection) {
			if content, exists := s.Attr("content"); exists && imageURL == "" {
				imageURL = content
			}
		})
	}
	if imageURL == "" {
		doc.Find("article img, div.section-media img").Each(func(i int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists && imageURL == "" {
				imageURL = src
			}
		})
	}

	// --- Scrape Description ---
	description := ""
	doc.Find("meta[property='og:description']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists && description == "" {
			description = strings.TrimSpace(content)
		}
	})
	if description == "" {
		doc.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
			if content, exists := s.Attr("content"); exists && description == "" {
				description = strings.TrimSpace(content)
			}
		})
	}
	if description == "" {
		doc.Find(".article__content p, .article-body p, .paragraph, .zn-body__paragraph").Each(func(i int, s *goquery.Selection) {
			if pText := strings.TrimSpace(s.Text()); len(pText) > 50 && description == "" {
				description = pText
			}
		})
	}

	if description != "" && len(description) > 200 {
		description = description[:200] + "..."
	}

	return imageURL, description, nil
}

// ServiceHealth is a simple exported function to satisfy Vercel's requirement
func ServiceHealth() string {
	return "News service is healthy"
} 