package handlers

import (
	"fmt"
	"net/http"
	"time"

	"gopkg.in/gin-gonic/gin.v1"

	"github.com/chrislewispac/realworld-starter-kit/models"
)

type Article struct {
	Slug           string   `json:"slug"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Body           string   `json:"body"`
	Favorited      bool     `json:"favorited"`
	FavoritesCount int      `json:"favoritesCount"`
	TagList        []string `json:"tagList"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
	Author         Author   `json:"user"`
}

type Author struct {
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	Image     string `json:"image"`
	Following bool   `json:"following"`
}

type ArticleJSON struct {
	Article `json:"article"`
}

type ArticlesJSON struct {
	Articles      []Article `json:"articles"`
	ArticlesCount int       `json:"articlesCount"`
}

func (h *Handler) extractArticle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if slug := c.Param("slug"); slug != "" {
			a, err := h.DB.GetArticle(slug)

			if err != nil {
				c.Abort()
				c.String(http.StatusNotFound, err.Error())
			}

			if a != nil {
				c.Set(fetchedArticleKey, a)
			}
		}
		h.Logger.Println("----------- extractArticle() -----------")
		c.Next()
	}
}

func (h *Handler) getArticle(c *gin.Context) {
	a := getFromContext(fetchedArticleKey, c).(*models.Article)

	u := getFromContext(currentUserKey, c).(*models.User)

	articleJSON := ArticleJSON{
		Article: h.buildArticleJSON(a, u),
	}

	c.JSON(200, articleJSON)
}

// getArticles handle GET /api/articles
func (h *Handler) getArticles(c *gin.Context) {
	var err error
	var articles = []models.Article{}
	c.Request.ParseForm()
	query := h.DB.GetAllArticles()

	query = h.DB.Limit(query, c.Request.Form)
	query = h.DB.Offset(query, c.Request.Form)
	query = h.DB.FilterByTag(query, c.Request.Form)
	query = h.DB.FilterAuthoredBy(query, c.Request.Form)
	query = h.DB.FilterFavoritedBy(query, c.Request.Form)

	err = query.Find(&articles).Error

	if err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	if len(articles) == 0 {
		c.JSON(http.StatusOK, ArticlesJSON{})
		return
	}

	u := getFromContext(currentUserKey, c).(*models.User)

	var articlesJSON ArticlesJSON
	for i := range articles {
		a := &articles[i]
		articlesJSON.Articles = append(articlesJSON.Articles, h.buildArticleJSON(a, u))
	}

	articlesJSON.ArticlesCount = len(articles)

	c.JSON(http.StatusOK, articlesJSON)
}

// createArticle handle POST /api/articles
func (h *Handler) createArticle(c *gin.Context) {
	var body struct {
		Article struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Body        string   `json:"body"`
			TagList     []string `json:"tagList"`
		} `json:"article"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	u := getFromContext(currentUserKey, c).(*models.User)
	a := models.NewArticle(body.Article.Title, body.Article.Description, body.Article.Body, u)

	if valid, errs := a.IsValid(); !valid {
		errorJSON := errorJSON{errs}
		c.JSON(http.StatusUnprocessableEntity, errorJSON)
		return
	}

	for _, tagName := range body.Article.TagList {
		tag, _ := h.DB.FindTagOrInit(tagName)
		a.Tags = append(a.Tags, tag)
	}

	if err := h.DB.CreateArticle(a); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	articleJSON := ArticleJSON{
		Article: h.buildArticleJSON(a, u),
	}

	c.JSON(http.StatusCreated, articleJSON)
}

// updateArticle handle PUT /api/articles/:slug
func (h *Handler) updateArticle(c *gin.Context) {
	var err error

	a := getFromContext(fetchedArticleKey, c).(*models.Article)
	u := getFromContext(currentUserKey, c).(*models.User)

	if !a.IsOwnedBy(u.Username) {
		err = fmt.Errorf("You don't have the permission to edit this article")
		c.String(http.StatusForbidden, err.Error())
		return
	}

	var body map[string]map[string]interface{}

	if err := c.BindJSON(&body); err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	if _, present := body["article"]; !present {
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	var article map[string]interface{}

	article = body["article"]

	if title, present := article["title"]; present {
		a.Title = title.(string)
	}

	if description, present := article["description"]; present {
		a.Description = description.(string)
	}

	if body, present := article["body"]; present {
		a.Body = body.(string)
	}

	if valid, errs := a.IsValid(); !valid {
		errorJSON := errorJSON{errs}
		c.JSON(http.StatusUnprocessableEntity, errorJSON)
		return
	}

	if err := h.DB.SaveArticle(a); err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	articleJSON := ArticleJSON{
		Article: h.buildArticleJSON(a, u),
	}

	c.JSON(http.StatusOK, articleJSON)
}

// deleteArticle handle DELETE /api/articles/:slug
func (h *Handler) deleteArticle(c *gin.Context) {
	var err error
	a := getFromContext(fetchedArticleKey, c).(*models.Article)
	u := getFromContext(currentUserKey, c).(*models.User)

	if !a.IsOwnedBy(u.Username) {
		err = fmt.Errorf("You don't have the permission to delete this article")
		c.String(http.StatusForbidden, err.Error())
		return
	}

	err = h.DB.DeleteArticle(a)

	if err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// favoriteArticle handle POST /api/articles/:slug/favorite
func (h *Handler) favoriteArticle(c *gin.Context) {
	a := getFromContext(fetchedArticleKey, c).(*models.Article)
	u := getFromContext(currentUserKey, c).(*models.User)

	err := h.DB.FavoriteArticle(u, a)

	articleJSON := ArticleJSON{
		Article: h.buildArticleJSON(a, u),
	}

	code := http.StatusOK

	if err != nil {
		code = http.StatusUnprocessableEntity
	}

	c.JSON(code, articleJSON)
}

// unFavoriteArticle handle DELETE /api/articles/:slug/favorite
func (h *Handler) unFavoriteArticle(c *gin.Context) {
	a := getFromContext(fetchedArticleKey, c).(*models.Article)
	u := getFromContext(currentUserKey, c).(*models.User)

	err := h.DB.UnfavoriteArticle(u, a)

	articleJSON := ArticleJSON{
		Article: h.buildArticleJSON(a, u),
	}

	code := http.StatusOK

	if err != nil {
		code = http.StatusUnprocessableEntity
	}

	c.JSON(code, articleJSON)
}

func (h *Handler) buildArticleJSON(a *models.Article, u *models.User) Article {
	following := false
	favorited := false

	if (u != &models.User{}) {
		following = h.DB.IsFollowing(u.ID, a.User.ID)
		favorited = h.DB.IsFavorited(u.ID, a.ID)
	}

	article := Article{
		Slug:           a.Slug,
		Title:          a.Title,
		Description:    a.Description,
		Body:           a.Body,
		Favorited:      favorited,
		FavoritesCount: a.FavoritesCount,
		CreatedAt:      a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      a.UpdatedAt.Format(time.RFC3339),
		Author: Author{
			Username:  a.User.Username,
			Bio:       a.User.Bio,
			Image:     a.User.Image,
			Following: following,
		},
	}

	for _, t := range a.Tags {
		article.TagList = append(article.TagList, t.Name)
	}

	return article
}
