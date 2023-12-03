package main

import (
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

	a := app.New()
	a.StartServer()

	log.Println("Application terminated!")
}
