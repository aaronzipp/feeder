package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/aaronzipp/feeder/database"
	"github.com/aaronzipp/feeder/tui"

	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	db, err := sql.Open("sqlite", "database/feeder.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	queries := database.New(db)

	if err := tui.Run(ctx, queries); err != nil {
		log.Fatal(err)
	}
}
