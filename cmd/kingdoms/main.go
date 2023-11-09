package main

import (
	"log"

	"kingdoms/internal/pkg/app"
)

// @title BITOP
// @version 1.0
// @description Bmstu Open IT Platform

// @contact.name API Support
// @contact.url https://vk.com/bmstu_schedule
// @contact.email bitop@spatecon.ru

// @license.name AS IS (NO WARRANTY)

// @host 127.0.0.1
// @schemes https http
// @BasePath /

func main() {
	log.Println("Application start!")

	a := app.New()
	a.StartServer()

	log.Println("Application terminated!")
}
