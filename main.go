package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	_ "modernc.org/sqlite"
)

var db *sql.DB
var templates *template.Template

// Initialize database
func initDB() {
	var err error
	// Check if database file exists
	_, err = os.Stat("productivity.db")
	dbExists := !os.IsNotExist(err)

	// Open database connection
	db, err = sql.Open("sqlite", "productivity.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create table if it doesn't exist
	if !dbExists {
		createTable := `
		CREATE TABLE productivity (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			productive INTEGER NOT NULL
		);`
		_, err = db.Exec(createTable)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Database initialized with productivity table")
	}
}

// Initialize templates
func initTemplates() {
	// Create templates directory if it doesn't exist
	if _, err := os.Stat("templates"); os.IsNotExist(err) {
		err = os.Mkdir("templates", 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	templates = template.Must(template.ParseGlob("templates/*.html"))
}

// Home page handler
func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		return
	}
}

// Check for missing entries and add default entries if needed
func checkMissingEntries() {
	// Get yesterday's date in YYYY-MM-DD format
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// Check if entry for yesterday exists
	var exists bool
	var id int
	err := db.QueryRow("SELECT id FROM productivity WHERE date = ?", yesterday).Scan(&id)
	exists = err == nil

	if !exists {
		// No entry for yesterday, insert default entry with productive=0
		_, err = db.Exec("INSERT INTO productivity (date, productive) VALUES (?, ?)", yesterday, 0)
		if err != nil {
			log.Printf("Failed to insert default entry for %s: %v", yesterday, err)
			return
		}
		log.Printf("Added default entry (non-productive) for %s", yesterday)
	}
}

// Record productivity handler
func recordProductivityHandler(w http.ResponseWriter, r *http.Request) {
	// Extract status from URL path using PathValue
	status := r.PathValue("status")

	var productive int
	if status == "productive" {
		productive = 1
	} else {
		productive = 0
	}

	// Get current date in YYYY-MM-DD format
	currentDate := time.Now().Format("2006-01-02")

	// Check if entry for current date already exists
	var exists bool
	var id int
	err := db.QueryRow("SELECT id FROM productivity WHERE date = ?", currentDate).Scan(&id)
	exists = err == nil

	var action string

	if exists {
		// Update existing record
		_, err = db.Exec("UPDATE productivity SET productive = ? WHERE date = ?", productive, currentDate)
		action = "updated"
	} else {
		// Insert new record
		_, err = db.Exec("INSERT INTO productivity (date, productive) VALUES (?, ?)", currentDate, productive)
		action = "inserted"
	}

	if err != nil {
		http.Error(w, "Failed to record productivity", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// Return success message
	w.Header().Set("Content-Type", "application/json")
	_, err = fmt.Fprintf(w, `{"status":"success", "message":"Productivity %s as %s"}`, action, status)
	if err != nil {
		return
	}
	log.Printf("Productivity %s as %s", action, status)
}

// Check if today has been entered
func checkToday(w http.ResponseWriter, r *http.Request) {
	// Get current date in YYYY-MM-DD format
	currentDate := time.Now().Format("2006-01-02")

	// Check if entry for current date already exists and get its status
	var exists bool
	var id int
	var productive int
	err := db.QueryRow("SELECT id, productive FROM productivity WHERE date = ?", currentDate).Scan(&id, &productive)
	exists = err == nil

	if exists {
		// Determine status string based on productive value
		status := "non-productive"
		if productive == 1 {
			status = "productive"
		}

		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintf(w, `{"status":"exists", "message":"Entry already exists for %s", "productive_status":"%s"}`, currentDate, status)
		if err != nil {
			return
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintf(w, `{"status":"not_exists", "message":"No entry for %s"}`, currentDate)
		if err != nil {
			return
		}
	}
}

// setupMidnightCheck sets up a scheduler to run checkMissingEntries at midnight
func setupMidnightCheck() {
	// Run the check immediately when the server starts
	checkMissingEntries()

	/*
		// Set up a goroutine to check at midnight every day
		go func() {
			for {
				now := time.Now()
				// Calculate time until next midnight
				nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
				timeUntilMidnight := nextMidnight.Sub(now)

				// Sleep until midnight
				log.Printf("Next check for missing entries scheduled at midnight (%s)", nextMidnight.Format("2006-01-02 15:04:05"))
				time.Sleep(timeUntilMidnight)

				// Run the check
				checkMissingEntries()
			}
		}()
	*/

	// Implementation using cron library
	c := cron.New()
	_, err := c.AddFunc("0 0 * * *", checkMissingEntries)
	if err != nil {
		return
	} // Run at midnight every day
	c.Start()

	log.Printf("Cron scheduler started. Next check for missing entries scheduled at midnight.")
}

func main() {
	// Initialize database
	initDB()
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	// Initialize templates
	initTemplates()

	// Set up midnight check for missing entries
	setupMidnightCheck()

	// Create router using standard library
	mux := http.NewServeMux()

	// Define routes with /ambition prefix
	mux.HandleFunc("/ambition/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ambition/" {
			http.NotFound(w, r)
			return
		}
		homeHandler(w, r)
	})

	mux.HandleFunc("GET /ambition/api/check-today", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		checkToday(w, r)
	})

	mux.HandleFunc("POST /ambition/api/record/{status}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		recordProductivityHandler(w, r)
	})

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/ambition/static/", http.StripPrefix("/ambition/static/", fs))

	// Create static directory if it doesn't exist
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		err = os.Mkdir("static", 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Start server
	log.Println("Server starting on http://localhost:3131")
	log.Fatal(http.ListenAndServe(":3131", mux))
}
