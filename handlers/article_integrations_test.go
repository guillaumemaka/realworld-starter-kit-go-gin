package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Machiel/slugify"
	"github.com/chrislewispac/realworld-starter-kit/auth"
	"github.com/chrislewispac/realworld-starter-kit/models"
	"github.com/jinzhu/gorm"
	"gopkg.in/testfixtures.v2"
)

type articleEntity struct {
	Article article `json:"article"`
}

type article struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Body        string   `json:"body"`
	TagList     []string `json:"tagList"`
}

var (
	h        *Handler
	DB       *gorm.DB
	articles []*models.Article
	fixtures *testfixtures.Context
)

func TestMain(m *testing.M) {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	db, err := models.NewDB("sqlite3", "../conduit_test.db")
	if err != nil {
		logger.Fatal(err)
	}

	DB = db.DB

	db.InitSchema()

	j := auth.NewJWT()
	h = New(db, j, logger)

	fixtures, err := testfixtures.NewFolder(DB.DB(), &testfixtures.SQLite{}, "../fixtures")
	if err != nil {
		log.Fatal(err)
	}

	if err := fixtures.Load(); err != nil {
		log.Fatal(err)
	}

	DB.Model(models.Article{}).
		Preload("User").
		Preload("Tags").
		Preload("Comments").
		Preload("Comments.User").
		Preload("Favorites").
		Preload("Favorites.User").
		Find(&articles)

	exit := m.Run()

	os.Exit(exit)
}

func TestArticlesHandler_Index(t *testing.T) {
	recorder := makeRequest(t, http.MethodGet, "/api/articles", nil, nil)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("should return a 200 status code: got %v want %v",
			status, http.StatusOK)
	}

	var articlesResponse ArticlesJSON
	json.NewDecoder(recorder.Body).Decode(&articlesResponse)

	expected := len(articles)
	if len(articlesResponse.Articles) != expected {
		t.Errorf("should return a list of articles: got %v want %v", len(articlesResponse.Articles), expected)
	}

	expectedUsername := articles[4].User.Username
	if article1 := articlesResponse.Articles[0]; article1.Author.Username != expectedUsername {
		t.Errorf("should return the correct author username: got %v want %v", article1.Author.Username, expectedUsername)
	}

	expectedUsername = articles[3].User.Username
	if article2 := articlesResponse.Articles[1]; article2.Author.Username != expectedUsername {
		t.Errorf("should return the correct author username: got %v want %v", article2.Author.Username, expectedUsername)
	}
}

func TestArticlesHandler_Read(t *testing.T) {
	a := articles[0]
	recorder := makeRequest(t, http.MethodGet, "/api/articles/"+a.Slug, nil, nil)

	var article ArticleJSON
	json.NewDecoder(recorder.Body).Decode(&article)

	if article.Article.Title != a.Title {
		t.Errorf("should return the correct article title: got %v want %v", article.Article.Title, "Title 5")
	}

	if article.Article.Description != a.Description {
		t.Errorf("should return the correct article description: got %v want %v", article.Article.Description, "Description 5")
	}

	if article.Article.Body != a.Body {
		t.Errorf("should return the correct article boy: got %v want %v", article.Article.Body, "Body 5")
	}

	if article.Article.Author.Username != a.User.Username {
		t.Errorf("should return the correct article author username: got %v want %v", article.Article.Author.Username, "user1")
	}
}

func TestArticlesHandler_FilterByTag(t *testing.T) {
	a := articles[0]
	recorder := makeRequest(t, http.MethodGet, "/api/articles?tag="+a.Tags[0].Name, nil, nil)

	var articlesResponse ArticlesJSON
	json.NewDecoder(recorder.Body).Decode(&articlesResponse)

	expectedLength := len(articles)
	if len(articlesResponse.Articles) != expectedLength {
		t.Errorf("should return the correct number article: got %v want %v", len(articlesResponse.Articles), expectedLength)
	}

	if article := articlesResponse.Articles[4]; article.Title != a.Title {
		t.Errorf("should return the correct article title: got %v want %v", article.Title, a.Title)
	}

	if article := articlesResponse.Articles[4]; article.Author.Username != a.User.Username {
		t.Errorf("should return the correct article author username: got %v want %v", article.Author.Username, a.User.Username)
	}
}

func TestArticlesHandler_FilterByAuthor(t *testing.T) {
	a := articles[0]
	recorder := makeRequest(t, http.MethodGet, "/api/articles?author="+a.User.Username, nil, nil)

	var articles ArticlesJSON
	json.NewDecoder(recorder.Body).Decode(&articles)

	if len(articles.Articles) != 1 {
		t.Errorf("should return the correct number article: got %v want %v", len(articles.Articles), 1)
	}

	if article := articles.Articles[0]; article.Author.Username != a.User.Username {
		t.Errorf("should return the correct article author username: got %v want %v", article.Author.Username, a.User.Username)
	}

	if article := articles.Articles[0]; article.Title != a.Title {
		t.Errorf("should return the correct article title: got %v want %v", article.Title, a.Title)
	}
}

func TestArticlesHandler_FilterByFavorited(t *testing.T) {
	a := articles[0]
	recorder := makeRequest(t, http.MethodGet, "/api/articles?favorited="+a.Favorites[0].User.Username, nil, nil)

	var articles ArticlesJSON
	json.NewDecoder(recorder.Body).Decode(&articles)

	if len(articles.Articles) != 5 {
		t.Errorf("should return the correct number article: got %v want %v", len(articles.Articles), 5)
	}

	if article := articles.Articles[4]; article.Title != a.Title {
		t.Errorf("should return the correct article title: got %v want %v", article.Title, a.Title)
	}
}

func TestArticlesHandler_CreateUnauthorized(t *testing.T) {
	a := Article{
		Title:       "GoLang Web Services",
		Description: "GoLang Web Services description",
		Body:        "GoLang Web Services",
		TagList:     []string{"Go"},
	}

	json, _ := json.Marshal(a)
	recorder := makeRequest(t, http.MethodPost, "/api/articles", bytes.NewBuffer(json), nil)

	if Code := recorder.Code; Code != http.StatusUnauthorized {
		t.Errorf("should return a 401 status code: got %v want %v", Code, http.StatusUnauthorized)
	}
}

func TestArticlesHandler_Create(t *testing.T) {
	a := articleEntity{
		Article: article{
			Title:       "GoLang Web Services",
			Description: "GoLang Web Services description",
			Body:        "GoLang Web Services",
			TagList:     []string{"Go", "Web Services"},
		},
	}

	var u = models.User{}
	DB.First(&u)
	jwt := auth.NewJWT().NewToken(u.Username)

	jsonBody, _ := json.Marshal(a)
	recorder := makeRequest(t, http.MethodPost, "/api/articles", bytes.NewBuffer(jsonBody), http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusCreated {
		t.Errorf("should return a 201 status code: got %v want %v", Code, http.StatusCreated)
	}

	var lastArticle = models.Article{}
	DB.Preload("Tags").Last(&lastArticle)

	var articleResponse ArticleJSON
	json.NewDecoder(recorder.Body).Decode(&articleResponse)

	if article := articleResponse.Article; article.Title != lastArticle.Title {
		t.Errorf("should return the correct article title: got %v want %v", article.Title, lastArticle.Title)
	}

	if article := articleResponse.Article; article.Description != lastArticle.Description {
		t.Errorf("should return the correct article description: got %v want %v", article.Description, lastArticle.Description)
	}

	if article := articleResponse.Article; article.Body != lastArticle.Body {
		t.Errorf("should return the correct article body: got %v want %v", article.Body, lastArticle.Body)
	}

	if article := articleResponse.Article; article.TagList[0] != lastArticle.Tags[0].Name {
		t.Errorf("should return the correct article tags: got %v want %v", article.TagList[0], lastArticle.Tags[0].Name)
	}

	if article := articleResponse.Article; article.TagList[1] != lastArticle.Tags[1].Name {
		t.Errorf("should return the correct article tags: got %v want %v", article.TagList[1], lastArticle.Tags[1].Name)
	}
}

func TestArticlesHandler_CreateWithEmptyTitle(t *testing.T) {
	a := articleEntity{
		Article: article{
			Title:       "",
			Description: "GoLang Web Services description",
			Body:        "GoLang Web Services",
			TagList:     []string{"Go", "Web Services"},
		},
	}

	jsonBody, _ := json.Marshal(a)

	var u = models.User{}
	DB.First(&u)

	jwt := auth.NewJWT().NewToken(u.Username)

	recorder := makeRequest(t, http.MethodPost, "/api/articles", bytes.NewBuffer(jsonBody), http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusUnprocessableEntity {
		t.Errorf("should return a 422 status code: got %v want %v", Code, http.StatusUnprocessableEntity)
	}

	var errorJSON errorJSON
	json.NewDecoder(recorder.Body).Decode(&errorJSON)

	if _, present := errorJSON.Errors["title"]; !present {
		t.Errorf("should return an error on the article title field: got %v want %v", present, true)
	}
}

func TestArticlesHandler_CreateWithEmptyDescription(t *testing.T) {
	a := articleEntity{
		Article: article{
			Title:       "GoLang Web Services",
			Description: "",
			Body:        "GoLang Web Services",
			TagList:     []string{"Go", "Web Services"},
		},
	}

	jsonBody, _ := json.Marshal(a)
	var u = models.User{}
	DB.First(&u)

	jwt := auth.NewJWT().NewToken(u.Username)

	recorder := makeRequest(t, http.MethodPost, "/api/articles", bytes.NewBuffer(jsonBody), http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusUnprocessableEntity {
		t.Errorf("should return a 422 status code: got %v want %v", Code, http.StatusUnprocessableEntity)
	}

	var errorJSON errorJSON
	json.NewDecoder(recorder.Body).Decode(&errorJSON)

	if _, present := errorJSON.Errors["description"]; !present {
		t.Errorf("should return an error on the article description field: got %v want %v", present, true)
	}
}

func TestArticlesHandler_CreateWithEmptyBody(t *testing.T) {
	a := articleEntity{
		Article: article{
			Title:       "GoLang Web Services",
			Description: "GoLang Web Services",
			Body:        "",
			TagList:     []string{"Go", "Web Services"},
		},
	}

	jsonBody, _ := json.Marshal(a)
	var u = models.User{}
	DB.First(&u)

	jwt := auth.NewJWT().NewToken(u.Username)

	recorder := makeRequest(t, http.MethodPost, "/api/articles", bytes.NewBuffer(jsonBody), http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusUnprocessableEntity {
		t.Errorf("should return a 422 status code: got %v want %v", Code, http.StatusUnprocessableEntity)
	}

	var errorJSON errorJSON
	json.NewDecoder(recorder.Body).Decode(&errorJSON)

	if _, present := errorJSON.Errors["body"]; !present {
		t.Errorf("should return an error on the article body field: got %v want %v", present, true)
	}
}

func TestArticlesHandler_UpdateForbidden(t *testing.T) {
	a := articles[0]
	var u = models.User{}
	DB.Last(&u)

	jsonBody, _ := json.Marshal(map[string]interface{}{
		"article": map[string]string{
			"title": "Title Should Not Be Updated",
		},
	})

	jwt := auth.NewJWT().NewToken(u.Username)

	recorder := makeRequest(t, http.MethodPut, "/api/articles/"+a.Slug, bytes.NewBuffer(jsonBody), http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusForbidden {
		t.Errorf("should return a 403 status code: got %v want %v", Code, http.StatusForbidden)
	}
}

func TestArticlesHandler_UpdateNotAuthorized(t *testing.T) {
	a := articles[0]

	jsonBody, _ := json.Marshal(map[string]interface{}{
		"article": map[string]string{
			"title": "Title Should Not Be Updated",
		},
	})

	recorder := makeRequest(t, http.MethodPut, "/api/articles/"+a.Slug, bytes.NewBuffer(jsonBody), nil)

	if Code := recorder.Code; Code != http.StatusUnauthorized {
		t.Errorf("should return a 401 status code: got %v want %v", Code, http.StatusUnauthorized)
	}
}

func TestArticlesHandler_UpdateOK(t *testing.T) {
	a := articles[0]
	updatedTitle := "Title Should Be Updated"

	jsonBody, _ := json.Marshal(map[string]interface{}{
		"article": map[string]string{
			"title": updatedTitle,
		},
	})

	jwt := auth.NewJWT().NewToken(a.User.Username)

	recorder := makeRequest(t, http.MethodPut, "/api/articles/"+a.Slug, bytes.NewBuffer(jsonBody), http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusOK {
		t.Errorf("should return a 200 status code: got %v want %v", Code, http.StatusOK)
	}

	var articleResponse ArticleJSON
	json.NewDecoder(recorder.Body).Decode(&articleResponse)

	article := articleResponse.Article
	if article.Title != updatedTitle {
		t.Errorf("should return the correct updated article title: got %v want %v", article.Title, updatedTitle)
	}

	updatedSlug := slugify.Slugify(updatedTitle)
	if article.Slug != updatedSlug {
		t.Errorf("should return the correct updated article slug: got %v want %v", article.Slug, updatedSlug)
	}

	DB.Save(&articles[0])
}

func TestArticlesHandler_Favorite(t *testing.T) {
	a := articles[0]
	u := articles[1].User

	jwt := auth.NewJWT().NewToken(u.Username)

	var recorder = makeRequest(t, http.MethodPost, "/api/articles/"+a.Slug+"/favorite", nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %v", jwt)},
	})

	var articleResponse ArticleJSON
	json.NewDecoder(recorder.Body).Decode(&articleResponse)

	if Code := recorder.Code; Code != http.StatusOK {
		t.Errorf("should get a 200 status code: got %v want %v", Code, http.StatusOK)
	}

	if articleResponse.Article.Favorited != true {
		t.Errorf("article should be in the state favorited: got %v want %v", articleResponse.Article.Favorited, true)
	}

	expectedCount := a.FavoritesCount + 1
	if articleResponse.Article.FavoritesCount != expectedCount {
		t.Errorf("article favorites count should be incremented by 1 : got %v want %v", articleResponse.Article.FavoritesCount, expectedCount)
	}

	DB.Save(&articles[0])
}

func TestArticlesHandler_FavoriteAlreadyFavoritedArticle(t *testing.T) {
	a := articles[0]
	u := articles[0].Favorites[0].User

	jwt := auth.NewJWT().NewToken(u.Username)

	var recorder = makeRequest(t, http.MethodPost, "/api/articles/"+a.Slug+"/favorite", nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %v", jwt)},
	})

	var articleResponse ArticleJSON
	json.NewDecoder(recorder.Body).Decode(&articleResponse)

	if Code := recorder.Code; Code != http.StatusUnprocessableEntity {
		t.Errorf("should get a 422 status code: got %v want %v", Code, http.StatusUnprocessableEntity)
	}

	if articleResponse.Article.Favorited != true {
		t.Errorf("article should be in the same state: got %v want %v", articleResponse.Article.Favorited, true)
	}
}

func TestArticlesHandler_Unfavorite(t *testing.T) {
	a := articles[0]
	u := articles[0].Favorites[0].User

	jwt := auth.NewJWT().NewToken(u.Username)

	var recorder = makeRequest(t, http.MethodDelete, "/api/articles/"+a.Slug+"/favorite", nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %v", jwt)},
	})

	var articleResponse ArticleJSON
	json.NewDecoder(recorder.Body).Decode(&articleResponse)

	if Code := recorder.Code; Code != http.StatusOK {
		t.Errorf("should get a 200 status code: got %v want %v", Code, http.StatusOK)
	}

	if articleResponse.Article.Favorited != false {
		t.Errorf("article should be in the state unfavorited: got %v want %v", articleResponse.Article.Favorited, false)
	}

	expectedCount := a.FavoritesCount - 1
	if articleResponse.Article.FavoritesCount != expectedCount {
		t.Errorf("article favorites count should be decremented by 1 : got %v want %v", articleResponse.Article.FavoritesCount, expectedCount)
	}
}

func TestArticlesHandler_UnfavoriteNotYetFavoritedArticle(t *testing.T) {
	a := articles[1]
	u := articles[2].User

	jwt := auth.NewJWT().NewToken(u.Username)

	var recorder = makeRequest(t, http.MethodDelete, "/api/articles/"+a.Slug+"/favorite", nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %v", jwt)},
	})

	var articleResponse ArticleJSON
	json.NewDecoder(recorder.Body).Decode(&articleResponse)

	if Code := recorder.Code; Code != http.StatusUnprocessableEntity {
		t.Errorf("should get a 422 status code: got %v want %v", Code, http.StatusUnprocessableEntity)
	}

	if articleResponse.Article.Favorited != false {
		t.Errorf("article should be in the same state: got %v want %v", articleResponse.Article.Favorited, false)
	}

	expectedCount := a.FavoritesCount
	if articleResponse.Article.FavoritesCount != expectedCount {
		t.Errorf("article favorites count should not be decremented by 1 : got %v want %v", articleResponse.Article.FavoritesCount, expectedCount)
	}
}

func TestArticlesHandler_DeleteOk(t *testing.T) {
	var u = &models.User{}
	DB.Last(&u)

	a := models.NewArticle("To Be Deleted", "Description", "Body", u)
	err := DB.Create(&a).Error

	if err != nil {
		t.Fatal(err)
	}

	jwt := auth.NewJWT().NewToken(u.Username)

	recorder := makeRequest(t, http.MethodDelete, "/api/articles/"+a.Slug, nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %v", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusNoContent {
		t.Errorf("should get a 204 status code: got %v want %v", Code, http.StatusNoContent)
	}
}

func TestArticlesHandler_DeleteForbidden(t *testing.T) {
	var author = &models.User{}
	DB.Last(&author)

	unauthorizedUser := articles[0].User

	a := models.NewArticle("Should Not Be Deleted", "Description", "Body", author)
	err := DB.Create(&a).Error

	if err != nil {
		t.Fatal(err)
	}

	jwt := auth.NewJWT().NewToken(unauthorizedUser.Username)

	recorder := makeRequest(t, http.MethodDelete, "/api/articles/"+a.Slug, nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %v", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusForbidden {
		t.Errorf("should get a 403 status code: got %v want %v", Code, http.StatusForbidden)
	}
}

func TestArticlesHandler_DeleteUnauthorized(t *testing.T) {
	slug := articles[0].Slug

	var recorder = makeRequest(t, http.MethodDelete, "/api/articles/"+slug, nil, nil)

	if Code := recorder.Code; Code != http.StatusUnauthorized {
		t.Errorf("should get a 401 status code: got %v want %v", Code, http.StatusUnauthorized)
	}
}

func TestArticlesHandler_ValidTokenButUserNotExist(t *testing.T) {
	jwt := auth.NewJWT().NewToken("non-existing-username")

	recorder := makeRequest(t, http.MethodGet, "/api/articles", nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %v", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusUnauthorized {
		t.Errorf("should get a 401 status code: got %v want %v", Code, http.StatusUnauthorized)
	}
}

///////////////////////////////////////////////////////////////////////////////
// Private & Helper Methods													 //
///////////////////////////////////////////////////////////////////////////////

func makeRequest(t *testing.T, method string, url string, body io.Reader, header http.Header) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		t.Fatal(err)
	}

	if header != nil {
		req.Header = header
	}

	var recorder = httptest.NewRecorder()

	router := h.InitRoutes()
	router.ServeHTTP(recorder, req)

	return recorder
}
