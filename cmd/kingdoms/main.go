package main

import (
	"context"
	"log"

	"kingdoms/internal/server/app"
)

// @title
// @version 1.0
// @description

// @contact.name API Support
// @contact.url
// @contact.email

// @host 127.0.0.1
// @schemes https http
// @BasePath /

func main() {
	log.Println("Application start!")

	a, err := app.New(context.Background())
	if err != nil {
		log.Println(err)

		return
	}

	a.StartServer()

	log.Println("Application terminated!")
}
