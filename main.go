package main

import (
	"log"
	"os"

	"github.com/chrislewispac/realworld-starter-kit/auth"
	"github.com/chrislewispac/realworld-starter-kit/handlers"
	"github.com/chrislewispac/realworld-starter-kit/models"
)

const (
	DATABASE string = "conduit.db"
	DIALECT  string = "sqlite3"
	PORT     string = ":8080"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	db, err := models.NewDB(DIALECT, DATABASE)
	if err != nil {
		logger.Fatal(err)
	}

	db.InitSchema()

	j := auth.NewJWT()
	h := handlers.New(db, j, logger)

	router := h.InitRoutes()

	router.Run(PORT)
}
