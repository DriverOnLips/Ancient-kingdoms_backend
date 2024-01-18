package main

import (
	api "Lab1/internal"
	"log"
)

func main() {
	log.Println("r start!")
	api.StartServer()
	log.Println("r terminated!")
}
