package handlers

import (
	"log"
	"net/http"

	"github.com/chrislewispac/realworld-starter-kit/auth"
	"github.com/chrislewispac/realworld-starter-kit/models"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"gopkg.in/gin-gonic/gin.v1"
)

type Handler struct {
	DB     models.Datastorer
	JWT    auth.Tokener
	Logger *log.Logger
}

type errorJSON struct {
	Errors models.ValidationErrors `json:"errors"`
}

const (
	currentUserKey    = "current_user"
	fetchedArticleKey = "article"
	claimKey          = "claim"
)

func New(db *models.DB, jwt *auth.JWT, logger *log.Logger) *Handler {
	return &Handler{db, jwt, logger}
}

func (h *Handler) authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		h.Logger.Println("----------- authorize() -----------")
		if claim, _ := c.Get(claimKey); claim != nil {
			if currentUser, ok := c.Get(currentUserKey); !ok && (currentUser == &models.User{}) {
				c.Abort()
				c.String(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			} else {
				c.Next()
			}
		} else {
			c.Abort()
			c.String(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		}
	}
}

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.Default()

	api := router.Group("/api")

	api.Use(h.getCurrentUser())
	api.GET("/articles", h.getArticles)
	api.POST("/articles", h.authorize(), h.createArticle)
	api.GET("/articles/:slug", h.extractArticle(), h.getArticle)
	api.PUT("/articles/:slug", h.authorize(), h.extractArticle(), h.updateArticle)
	api.DELETE("/articles/:slug", h.authorize(), h.extractArticle(), h.deleteArticle)

	api.GET("/articles/:slug/comments", h.extractArticle(), h.getComments)
	api.POST("/articles/:slug/comments", h.authorize(), h.extractArticle(), h.addComment)
	api.GET("/articles/:slug/comments/:commentID", h.extractArticle(), h.getComment)
	api.DELETE("/articles/:slug/comments/:commentID", h.authorize(), h.extractArticle(), h.deleteComment)

	api.POST("/articles/:slug/favorite", h.authorize(), h.extractArticle(), h.favoriteArticle)
	api.DELETE("/articles/:slug/favorite", h.authorize(), h.extractArticle(), h.unFavoriteArticle)
	api.GET("/users", h.currentUser)
	api.POST("/users", h.registerUser)
	api.POST("/users/login", h.loginUser)

	return router
}

func getFromContext(key string, c *gin.Context) interface{} {
	obj, _ := c.Get(key)
	return obj
}
