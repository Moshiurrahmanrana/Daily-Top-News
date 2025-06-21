package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly/v2"
)

// NewsService handles news fetching operations
type NewsService struct {
	sources map[string]Source
	client  *http.Client
}

// NewNewsService creates a new news service instance
func NewNewsService() *NewsService {
	// Initialize news sources - only The Daily Star
	sources := map[string]Source{
		"thedailystar": {
			Name:        "thedailystar",
			DisplayName: "The Daily Star",
			URL:         "https://www.thedailystar.net/",
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
	allNews := make(chan []NewsArticle, len(ns.sources))
	
	// Fetch news from all sources concurrently
	for name, source := range ns.sources {
		if !source.Active {
			continue
		}
		
		wg.Add(1)
		go func(sourceName string, source Source) {
			defer wg.Done()
			news, err := ns.fetchNewsFromSource(sourceName, source.URL)
			if err != nil {
				log.Printf("Error fetching from %s: %v", sourceName, err)
				allNews <- []NewsArticle{}
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
	var allArticles []NewsArticle
	for news := range allNews {
		allArticles = append(allArticles, news...)
	}

	response := NewsResponse{
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
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error:   "source_not_found",
			Message: "News source not found",
		})
		return
	}

	if !source.Active {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "source_inactive",
			Message: "News source is currently inactive",
		})
		return
	}

	news, err := ns.fetchNewsFromSource(sourceName, source.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "fetch_error",
			Message: fmt.Sprintf("Failed to fetch news: %v", err),
		})
		return
	}

	response := NewsResponse{
		Success: true,
		Data:    news,
		Count:   len(news),
		Source:  sourceName,
	}

	c.JSON(http.StatusOK, response)
}

// GetAvailableSources returns all available news sources
func (ns *NewsService) GetAvailableSources(c *gin.Context) {
	var sources []Source
	for _, source := range ns.sources {
		sources = append(sources, source)
	}

	response := SourcesResponse{
		Success: true,
		Sources: sources,
	}

	c.JSON(http.StatusOK, response)
}

// fetchNewsFromSource fetches news from a specific source
func (ns *NewsService) fetchNewsFromSource(sourceName, url string) ([]NewsArticle, error) {
	// Only handle The Daily Star
	if sourceName == "thedailystar" {
		return ns.fetchTheDailyStarWithColly(url)
	}

	return nil, fmt.Errorf("unsupported source: %s", sourceName)
}

// fetchTheDailyStarWithColly fetches news from The Daily Star using Colly
func (ns *NewsService) fetchTheDailyStarWithColly(url string) ([]NewsArticle, error) {
	// Initialize a slice to store articles
	articles := []NewsArticle{}

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
		article := NewsArticle{
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
	ns.updateMissingImageURLs(&articles)

	return articles, nil
}

// updateMissingImageURLs updates empty image_url fields by scraping from the article URL
func (ns *NewsService) updateMissingImageURLs(articles *[]NewsArticle) {
	for i := range *articles {
		if (*articles)[i].ImageURL == "" {
			imageURL, err := ns.scrapeImageFromURL((*articles)[i].URL)
			if err != nil {
				log.Printf("Error scraping image: %v", err)
				continue
			}
			(*articles)[i].ImageURL = imageURL
			// Add delay to avoid overwhelming the server
			time.Sleep(1 * time.Second)
		}
	}
}

// scrapeImageFromURL fetches an image URL from the given webpage
func (ns *NewsService) scrapeImageFromURL(url string) (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set User-Agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse HTML using goquery
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Try to find image in <picture> tag (data-srcset)
	imageURL := ""
	doc.Find("picture img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("data-srcset"); exists && imageURL == "" {
			imageURL = src
		}
	})
	if imageURL != "" {
		return imageURL, nil
	}

	// Fallback: Try to find image in lg-gallery span (data-src)
	doc.Find("span.lg-gallery").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("data-src"); exists && imageURL == "" {
			imageURL = src
		}
	})
	if imageURL != "" {
		return imageURL, nil
	}

	// Fallback: Try to find Open Graph image
	doc.Find("meta[property='og:image']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists && imageURL == "" {
			imageURL = content
		}
	})
	if imageURL != "" {
		return imageURL, nil
	}

	// Fallback: Find the first image in the article body
	doc.Find("article img, div.section-media img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && imageURL == "" {
			imageURL = src
		}
	})

	if imageURL != "" {
		return imageURL, nil
	}

	return "", fmt.Errorf("no image found on page: %s", url)
} 