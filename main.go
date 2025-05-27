/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/shah1011/obscure/cmd"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  .env file not found or couldn't be loaded. Falling back to default env vars.")
	}
	cmd.Execute()
}
