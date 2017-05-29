package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/chrislewispac/realworld-starter-kit/models"
	"gopkg.in/gin-gonic/gin.v1"
)

type Comment struct {
	ID        int    `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Author    Author `json:"author"`
}

type CommentJSON struct {
	Comment `json:"comment"`
}

type CommentsJSON struct {
	Comments []Comment `json:"comments"`
}

type commentBody struct {
	Comment struct {
		Body string `json:"body"`
	} `json:"comment"`
}

func (h *Handler) getComments(c *gin.Context) {
	a := getFromContext(fetchedArticleKey, c).(*models.Article)
	u := getFromContext(currentUserKey, c).(*models.User)

	var comments []models.Comment
	err := h.DB.GetComments(a, &comments)

	if err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	var commentsJSON = CommentsJSON{}
	for _, comment := range comments {
		commentsJSON.Comments = append(commentsJSON.Comments, h.buildCommentJSON(&comment, u))
	}

	c.JSON(http.StatusOK, commentsJSON)
}

func (h *Handler) getComment(c *gin.Context) {
	commentID, _ := strconv.Atoi(c.Param("commentID"))
	u := getFromContext(currentUserKey, c).(*models.User)

	var comment models.Comment
	err := h.DB.GetComment(commentID, &comment)

	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	var commentJSON = CommentJSON{
		Comment: h.buildCommentJSON(&comment, u),
	}

	c.JSON(http.StatusOK, commentJSON)
}

func (h *Handler) addComment(c *gin.Context) {
	a := getFromContext(fetchedArticleKey, c).(*models.Article)
	u := getFromContext(currentUserKey, c).(*models.User)

	var commentBody commentBody
	if err := c.BindJSON(&commentBody); err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	newComment, errs := models.NewComment(a, u, commentBody.Comment.Body)

	if errs != nil {
		errorJSON := errorJSON{errs}

		c.JSON(http.StatusUnprocessableEntity, errorJSON)
		return
	}

	err := h.DB.CreateComment(newComment)

	if err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	commentJSON := CommentJSON{
		Comment: h.buildCommentJSON(newComment, u),
	}

	c.JSON(http.StatusCreated, commentJSON)
}

func (h *Handler) deleteComment(c *gin.Context) {
	commentID, _ := strconv.Atoi(c.Param("commentID"))
	u := getFromContext(currentUserKey, c).(*models.User)

	var comment = models.Comment{}
	err := h.DB.GetComment(commentID, &comment)

	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	if canDelete := comment.CanBeDeletedBy(u); !canDelete {
		c.String(http.StatusForbidden, http.StatusText(http.StatusForbidden))
		return
	}

	err = h.DB.DeleteComment(&comment)

	if err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	c.String(http.StatusNoContent, http.StatusText(http.StatusNoContent))
}

func (h *Handler) buildCommentJSON(c *models.Comment, u *models.User) Comment {
	following := false

	if (u != &models.User{}) {
		following = h.DB.IsFollowing(u.ID, c.User.ID)
	}

	return Comment{
		ID:        c.ID,
		Body:      c.Body,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		Author: Author{
			Username:  c.User.Username,
			Bio:       c.User.Bio,
			Image:     c.User.Image,
			Following: following,
		},
	}
}
