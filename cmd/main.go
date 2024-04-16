package main

import (
	"context"
	"example.com/medium/ent"
	"example.com/medium/ent/migrate"
	"example.com/medium/internal/mediumapp"
	"fmt"
	"os"
	_ "github.com/lib/pq"
	"log"
)

func main() {
	dbPass := os.Getenv("DB_PASSWORD")
	client, err := ent.Open("postgres", fmt.Sprintf("host=localhost port=5432 user=myuser dbname=realdb password=%s", dbPass))
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer client.Close()
	// Run the auto migration tool.
	if err := client.Schema.Create(
		context.Background(),
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true)); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
	mediumapp.StartServer(client)
}
