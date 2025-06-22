package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var router *gin.Engine

func init() {
	router = setupRouter()
}

// Handler is the entry point for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	router.ServeHTTP(w, r)
} 