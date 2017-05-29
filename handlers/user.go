package handlers

import (
	"fmt"
	"net/http"

	"github.com/guillaumemaka/realworld-starter-kit-go-gin/models"
	"gopkg.in/gin-gonic/gin.v1"
)

// User is the user json object for responses
type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Token    string `json:"token"`
	Bio      string `json:"bio"`
	Image    string `json:"image"`
}

// UserJSON is the wrapper around User to give it a key "user"
type UserJSON struct {
	User *User `json:"user"`
}

// getCurrentUser is a middleware that extracts the current user into context
func (h *Handler) getCurrentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		var u = &models.User{}

		if claim, _ := h.JWT.CheckRequest(c.Request); claim != nil {
			// Check also that user exists and prevent old token usage
			// to gain privillege access.
			if u, err = h.DB.FindUserByUsername(claim.Username); err != nil {
				c.Abort()
				c.String(http.StatusUnauthorized, fmt.Sprint("User with username", claim.Username, "doesn't exist !"))
			}
			c.Set(claimKey, claim)
		}

		c.Set(currentUserKey, u)
		c.Next()
	}
}

// POST /user
// regiesterUser adds a new user to the database and response with the new user
func (h *Handler) registerUser(c *gin.Context) {
	body := struct {
		User struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"user"`
	}{}
	bodyUser := &body.User

	err := c.BindJSON(&bodyUser)
	if err != nil {
		return
	}

	u, errs := models.NewUser(bodyUser.Email, bodyUser.Username, bodyUser.Password)
	if errs != nil {
		errorJSON := errorJSON{errs}
		c.JSON(http.StatusUnprocessableEntity, errorJSON)
		return
	}

	err = h.DB.CreateUser(u)
	if err != nil {
		// TODO: Error JSON
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	res := &UserJSON{
		&User{
			Username: u.Username,
			Email:    u.Email,
			Token:    h.JWT.NewToken(u.Username),
		},
	}

	c.JSON(http.StatusOK, res)
}

// POST /user/login
// loginUser returns an user according to the credentials provided
func (h *Handler) loginUser(c *gin.Context) {
	body := struct {
		User struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"user"`
	}{}
	bodyUser := &body.User

	err := c.BindJSON(&bodyUser)
	if err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	u, err := h.DB.FindUserByEmail(bodyUser.Email)
	if err != nil {
		// TODO: Error JSON
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	match := u.MatchPassword(bodyUser.Password)
	if !match {
		// TODO: Error JSON
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	res := &UserJSON{
		&User{
			Username: u.Username,
			Email:    u.Email,
			Token:    h.JWT.NewToken(u.Username),
			Bio:      u.Bio,
			Image:    u.Image,
		},
	}

	c.JSON(http.StatusOK, res)
}

// GET /user
// currentUser responds with the current user
func (h *Handler) currentUser(c *gin.Context) {
	u := getFromContext(currentUserKey, c).(*models.User)

	res := &UserJSON{
		&User{
			Username: u.Username,
			Email:    u.Email,
			// TODO: Use same token that was provided?
			Token: h.JWT.NewToken(u.Username),
			Bio:   u.Bio,
			Image: u.Image,
		},
	}

	c.JSON(http.StatusOK, res)
}
