package config

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load("../.env")
	if err != nil {
		// Try current directory as fallback (for when running from root/)
		err = godotenv.Load()
		if err != nil {
			log.Println("No .env file found, using system environment variables")
		}
	}
}
