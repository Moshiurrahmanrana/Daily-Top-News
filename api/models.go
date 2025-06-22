package main

import "time"

// NewsArticle represents a single news article
type NewsArticle struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ImageURL    string    `json:"image_url"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	Category    string    `json:"category,omitempty"`
}

// NewsResponse represents the API response for news
type NewsResponse struct {
	Success bool          `json:"success"`
	Data    []NewsArticle `json:"data"`
	Count   int           `json:"count"`
	Source  string        `json:"source,omitempty"`
}

// SourcesResponse represents the API response for available sources
type SourcesResponse struct {
	Success bool     `json:"success"`
	Sources []Source `json:"sources"`
}

// Source represents a news source configuration
type Source struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
	Active      bool   `json:"active"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
} 