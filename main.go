package main

import (
	"log"

	binance "binance-api/binance"

	"github.com/joho/godotenv"
)

var (
	apiKey    = ""
	secretKey = ""
)

func main() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	binance.ListenCoins()

}
