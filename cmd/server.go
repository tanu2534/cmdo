package cmd

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanu2534/cmdo/database"
)

// Command struct for JSON response
type CommandJSON struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	ExitCode  int       `json:"exitCode"`
	Timestamp time.Time `json:"timestamp"`
	Folder    string    `json:"folder"`
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI server",
	Long:  "Starts a local web server to view command history",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")

		database.InitDB("./cmdo.db")
		defer database.DB.Close()

		// API endpoints
		http.HandleFunc("/api/commands", apiCommandsHandler)
		http.HandleFunc("/api/delete", apiDeleteHandler)
		http.HandleFunc("/api/clear", apiClearHandler)

		// Serve HTML page
		http.HandleFunc("/", indexHandler)

		fmt.Printf("🚀 CMDO Server running at http://localhost:%s\n", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	},
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("server/index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func apiCommandsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	log.Printf("apiCommandsHandler: Received request from %s", r.RemoteAddr)

	rows, err := database.DB.Query(`
		SELECT id, command, exit_code, timestamp, directory 
		FROM commands 
		ORDER BY timestamp DESC
	`)
	if err != nil {
		log.Printf("apiCommandsHandler: Error querying database: %s", err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var commands []CommandJSON
	for rows.Next() {
		var cmd CommandJSON
		var ts string

		err := rows.Scan(&cmd.ID, &cmd.Command, &cmd.ExitCode, &ts, &cmd.Folder)
		if err != nil {
			log.Printf("apiCommandsHandler: Error scanning row: %s", err)
			continue
		}

		parsedTime, err := time.Parse("2006-01-02 15:04:05", ts)
		if err != nil {
			log.Printf("apiCommandsHandler: Error parsing time: %s", err)
			continue
		}

		cmd.Timestamp = parsedTime
		commands = append(commands, cmd)
	}

	log.Printf("apiCommandsHandler: Returning %d commands", len(commands))
	json.NewEncoder(w).Encode(commands)
}

func apiDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	_, err := database.DB.Exec("DELETE FROM commands WHERE id = ?", req.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func apiClearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	_, err := database.DB.Exec("DELETE FROM commands")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func init() {
	serveCmd.Flags().String("port", "8089", "Port to run the server on")
	rootCmd.AddCommand(serveCmd)
}
