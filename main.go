package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
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
	db, err = sql.Open("sqlite3", "productivity.db")
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
	templates.ExecuteTemplate(w, "index.html", nil)
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
	vars := mux.Vars(r)
	status := vars["status"]

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
	fmt.Fprintf(w, `{"status":"success", "message":"Productivity %s as %s"}`, action, status)
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
	defer db.Close()

	// Initialize templates
	initTemplates()

	// Set up midnight check for missing entries
	setupMidnightCheck()

	// Create router
	r := mux.NewRouter()

	// Create a subrouter with the /ambition prefix
	s := r.PathPrefix("/ambition").Subrouter()

	// Define routes
	s.HandleFunc("/", homeHandler).Methods("GET")
	s.HandleFunc("/api/record/{status}", recordProductivityHandler).Methods("POST")

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	s.PathPrefix("/static/").Handler(http.StripPrefix("/ambition/static/", fs))

	// Create static directory if it doesn't exist
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		err = os.Mkdir("static", 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Start server
	log.Println("Server starting on http://localhost:3131")
	log.Fatal(http.ListenAndServe(":3131", r))
}
