package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/guillaumemaka/realworld-starter-kit-go-gin/models"
)

func Test_GetComments(t *testing.T) {
	expectedCount := DB.Model(articles[0]).Association("Comments").Count()
	recorder := makeRequest(t, http.MethodGet, "/api/articles/"+articles[0].Slug+"/comments", nil, nil)
	var commentsResponse CommentsJSON
	json.NewDecoder(recorder.Body).Decode(&commentsResponse)

	if Code := recorder.Code; Code != http.StatusOK {
		t.Errorf("should return a 200 status code: got %v want %v", Code, http.StatusOK)
	}

	if count := len(commentsResponse.Comments); count != expectedCount {
		t.Errorf("should return a the correct number of comment for the given article status code: got %v want %v", count, expectedCount)
	}

	for _, comment := range commentsResponse.Comments {
		if empty := (comment.Author.Username == ""); empty {
			t.Errorf("should contain the author username : got %v want %v", empty, false)
		}

		if empty := (comment.Author.Bio == ""); empty {
			t.Errorf("should contain the author bio : got %v want %v", empty, false)
		}

		if empty := (comment.Author.Image == ""); empty {
			t.Errorf("should contain the author image : got %v want %v", empty, false)
		}
	}
}

func Test_GetComment(t *testing.T) {
	recorder := makeRequest(t, http.MethodGet, "/api/articles/"+articles[0].Slug+"/comments/1", nil, nil)

	var commentResponse CommentJSON
	json.NewDecoder(recorder.Body).Decode(&commentResponse)

	if Code := recorder.Code; Code != http.StatusOK {
		t.Errorf("should return a 200 status code: got %v want %v", Code, http.StatusOK)
	}

	expectedCommentID := articles[0].Comments[0].ID
	if commentResponse.Comment.ID != expectedCommentID {
		t.Errorf("should get the correct author: got %v want %v", commentResponse.Comment.ID, expectedCommentID)
	}

	expectedCommentBody := articles[0].Comments[0].Body
	if commentResponse.Comment.Body != expectedCommentBody {
		t.Errorf("should get the correct author: got %v want %v", commentResponse.Comment.Body, expectedCommentBody)
	}

	expectedCommentAuthor := articles[0].Comments[0].User.Username
	if commentResponse.Comment.Author.Username != expectedCommentAuthor {
		t.Errorf("should get the correct author: got %v want %v", commentResponse.Author.Username, expectedCommentAuthor)
	}
}

func Test_GetCommentNotFound(t *testing.T) {
	recorder := makeRequest(t, http.MethodGet, "/api/articles/"+articles[0].Slug+"/comments/1000", nil, nil)

	var commentResponse CommentJSON
	json.NewDecoder(recorder.Body).Decode(&commentResponse)

	if Code := recorder.Code; Code != http.StatusNotFound {
		t.Errorf("should return a 404 status code: got %v want %v", Code, http.StatusNotFound)
	}
}

func Test_PostCommentOK(t *testing.T) {
	commentBody := map[string]interface{}{
		"comment": map[string]string{
			"body": "Comment Body",
		},
	}

	a := articles[2]

	var u = models.User{}
	DB.First(&u)

	jwt := h.JWT.NewToken(u.Username)
	jsonBody, _ := json.Marshal(commentBody)

	recorder := makeRequest(t, http.MethodPost, "/api/articles/"+a.Slug+"/comments", bytes.NewBuffer(jsonBody), http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusCreated {
		t.Errorf("should return a 201 status code: got %v want %v", Code, http.StatusCreated)
	}

	var commentResponse CommentJSON
	json.NewDecoder(recorder.Body).Decode(&commentResponse)

	var lastComment = models.Comment{}
	DB.Preload("User").Last(&lastComment)

	if commentResponse.Comment.ID != lastComment.ID {
		t.Errorf("should return the correct comment id: got %v want %v", commentResponse.Comment.ID, lastComment.ID)
	}

	if commentResponse.Comment.Body != lastComment.Body {
		t.Errorf("should return the correct comment body: got %v want %v", commentResponse.Comment.Body, lastComment.Body)
	}

	if commentResponse.Comment.Author.Username != lastComment.User.Username {
		t.Errorf("should return the correct comment author username: got %v want %v", commentResponse.Comment.Author.Username, lastComment.User.Username)
	}

	if commentResponse.Comment.Author.Username != u.Username {
		t.Errorf("comment Author username must be the same as the loggedin user: got %v want %v", commentResponse.Comment.Author.Username, u.Username)
	}
}

func Test_PostCommentEmptyBody(t *testing.T) {
	commentBody := map[string]interface{}{
		"comment": map[string]string{
			"body": "",
		},
	}

	a := articles[0]

	var u = models.User{}
	DB.First(&u)

	jwt := h.JWT.NewToken(u.Username)
	jsonBody, _ := json.Marshal(commentBody)

	recorder := makeRequest(t, http.MethodPost, "/api/articles/"+a.Slug+"/comments", bytes.NewBuffer(jsonBody), http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusUnprocessableEntity {
		t.Errorf("should return a 422 status code: got %v want %v", Code, http.StatusUnprocessableEntity)
	}

	var errorJSON errorJSON
	json.NewDecoder(recorder.Body).Decode(&errorJSON)

	errorMsg, ok := errorJSON.Errors["body"]

	if !ok {
		t.Errorf("should return an error on the body field: got %v want %v", ok, true)
	}

	if errorMsg[0] != models.EMPTY_MSG {
		t.Errorf("should return an error message on the body field: got %v want %v", errorMsg[0], models.EMPTY_MSG)
	}
}

func Test_PostCommentUnauthorized(t *testing.T) {
	commentBody := map[string]interface{}{
		"comment": map[string]string{
			"body": "Comment Body",
		},
	}

	a := articles[2]

	jsonBody, _ := json.Marshal(commentBody)

	recorder := makeRequest(t, http.MethodPost, "/api/articles/"+a.Slug+"/comments", bytes.NewBuffer(jsonBody), nil)

	if Code := recorder.Code; Code != http.StatusUnauthorized {
		t.Errorf("should return a 401 status code: got %v want %v", Code, http.StatusUnauthorized)
	}
}

func Test_DeleteCommentNotFound(t *testing.T) {
	jwt := h.JWT.NewToken("user1")

	recorder := makeRequest(t, http.MethodDelete, "/api/articles/"+articles[2].Slug+"/comments/1000", nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusNotFound {
		t.Errorf("should return a 404 status code: got %v want %v", Code, http.StatusNotFound)
	}
}

func Test_DeleteCommentForbidden(t *testing.T) {
	jwt := h.JWT.NewToken("user1")

	recorder := makeRequest(t, http.MethodDelete, "/api/articles/"+articles[2].Slug+"/comments/1", nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusForbidden {
		t.Errorf("should return a 403 status code: got %v want %v", Code, http.StatusForbidden)
	}

	err := DB.First(&Comment{ID: 1}).Error
	if got := (err != nil); got {
		t.Errorf("should not delete the comment got %v want %v", got, false)
	}
}

func Test_DeleteCommentOK(t *testing.T) {
	article := articles[0]
	commentID := article.Comments[0].ID
	author := article.Comments[0].User.Username

	jwt := h.JWT.NewToken(author)

	recorder := makeRequest(t, http.MethodDelete, "/api/articles/"+article.Slug+"/comments/"+strconv.Itoa(commentID), nil, http.Header{
		"Authorization": []string{fmt.Sprintf("Token %s", jwt)},
	})

	if Code := recorder.Code; Code != http.StatusNoContent {
		t.Errorf("should return a 204 status code: got %v want %v", Code, http.StatusNoContent)
	}

	err := DB.First(&Comment{ID: commentID}).Error
	if got := (err == nil); got {
		t.Errorf("should delete the comment got %v want %v", got, true)
	}
}
