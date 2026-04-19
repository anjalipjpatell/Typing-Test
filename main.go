package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

const port = "3333"

func initDB(dbPath string) (*sql.DB, error) {
	// Ensure the directory for the database exists
	if err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS words (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		word TEXT NOT NULL UNIQUE
	);`
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, err
	}

	return db, nil
}

func populateDatabase(db *sql.DB, wordsFilePath string) error {
	// Check if the database is already populated
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM words").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		fmt.Printf("Database already populated with %d words.\n", count)
		return nil
	}

	// Open the text file
	file, err := os.Open(wordsFilePath)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", wordsFilePath, err)
	}
	defer file.Close()

	// Start a transaction for bulk inserts (drastically speeds up SQLite)
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO words (word) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(file)
	inserted := 0
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			if _, err := stmt.Exec(word); err != nil {
				tx.Rollback()
				return err
			}
			inserted++
		}
	}

	if err := scanner.Err(); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("Successfully populated database with %d words.\n", inserted)
	return nil
}

func main() {
	// Initialize the database inside a "data" folder
	db, err := initDB("./data/typing_test.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Populate database with words from text file
	if err := populateDatabase(db, "./words.txt"); err != nil {
		log.Fatalf("Failed to populate database: %v", err)
	}

	// Serve static files from frontend/dist directory
	fs := http.FileServer(http.Dir("./frontend/dist"))
	http.Handle("/", fs)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
	}

	// Start the server in a goroutine so it doesn't block the main thread
	go func() {
		fmt.Printf("Server starting on http://localhost:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Create a channel to listen for OS signals (like Ctrl+C), and block until a signal is recieved
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	fmt.Println("\nShutting down server...")

	// Give the server 3 seconds to finish any ongoing requests
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exited gracefully")
}
