package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// NewsArticle represents a news article
type NewsArticle struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ImageURL    string    `json:"image_url"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
}

// Response represents the JSON output structure
type Response struct {
	Success bool          `json:"success"`
	Data    []NewsArticle `json:"data"`
	Count   int           `json:"count"`
	Source  string        `json:"source"`
}

func main() {
	// Initialize response
	response := Response{
		Success: true,
		Data:    []NewsArticle{},
		Count:   0,
		Source:  "thedailystar",
	}

	// Create a new Colly collector
	c := colly.NewCollector(
		colly.AllowedDomains("www.thedailystar.net"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	// Counter for article IDs
	articleID := 0

	// OnHTML callback to extract article data
	c.OnHTML("article, .article, .news-item, .story, .post, .card, .item, div[class*='article'], div[class*='news'], div[class*='story']", func(e *colly.HTMLElement) {
		var title, link string

		// Try different title selectors - get only the first/main title
		titleSelectors := []string{
			"h1", "h2", "h3", "h4",
			".title", ".headline",
			"h1 a", "h2 a", "h3 a", "h4 a",
			".title a", ".headline a",
			"a[class*='title']", "a[class*='headline']",
			".article-title a", ".news-title a",
		}
		
		for _, selector := range titleSelectors {
			title = strings.TrimSpace(e.ChildText(selector))
			link = e.ChildAttr(selector, "href")
			
			if title != "" && link != "" {
				break
			}
			
			// If no link found, try to find any link within the element
			if title != "" && link == "" {
				link = e.ChildAttr("a", "href")
				if link != "" {
					break
				}
			}
		}

		// Skip if title or link is empty
		if title == "" || link == "" {
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

		// Ensure the link is absolute
		link = e.Request.AbsoluteURL(link)

		// Extract image URL
		imageURL := e.ChildAttr("img", "src")
		if imageURL == "" {
			imageURL = e.ChildAttr("img", "data-src")
		}
		if imageURL == "" {
			imageURL = e.ChildAttr("img", "data-lazy-src")
		}
		imageURL = e.Request.AbsoluteURL(imageURL)

		// Create NewsArticle struct
		article := NewsArticle{
			ID:          fmt.Sprintf("dailystar_%d", articleID),
			Title:       title,
			Description: "", // Placeholder, as descriptions may not be on homepage
			ImageURL:    imageURL,
			URL:         link,
			Source:      "thedailystar",
			PublishedAt: time.Now(),
		}

		// Append to response
		response.Data = append(response.Data, article)
		response.Count++
		articleID++
	})

	// OnError callback to handle errors
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error:", err, "Status Code:", r.StatusCode)
		response.Success = false
	})

	// Start scraping the homepage
	err := c.Visit("https://www.thedailystar.net/")
	if err != nil {
		fmt.Println("Failed to visit:", err)
		response.Success = false
	}

	// Wait for all requests to complete
	c.Wait()

	// Output JSON response
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Print to console
	fmt.Println(string(jsonData))
} 