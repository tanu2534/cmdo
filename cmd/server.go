package cmd

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanu2534/cmdo/database"
)

//go:embed server/index.html
var indexHTML embed.FS

// Command struct for JSON response
// Command struct for JSON response - YE CHANGE KARO
type CommandJSON struct {
	ID        string `json:"id"`
	Command   string `json:"command"`
	ExitCode  int    `json:"exitCode"`
	Timestamp string `json:"timestamp"` // TIME.TIME KI JAGAH STRING
	Folder    string `json:"folder"`
}

type StatsJSON struct {
	TotalCommands   int    `json:"totalCommands"`
	OldestTimestamp string `json:"oldestTimestamp"`
	NewestTimestamp string `json:"newestTimestamp"`
	DatabaseSize    string `json:"databaseSize"`
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI server",
	Long:  "Starts a local web server to view command history",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")

		// Use global DB path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Error getting home directory:", err)
		}
		dbPath := filepath.Join(homeDir, ".cmdo", "cmdo.db")

		// Check if DB exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Println("No database found at:", dbPath)
			fmt.Println("Run some commands first, or use 'cmdo setup' to install hooks")
		}

		database.InitDB(dbPath)
		defer database.DB.Close()

		// Generate session token
		token := generateToken()
		tokenFile := filepath.Join(homeDir, ".cmdo", ".session")
		os.WriteFile(tokenFile, []byte(token), 0600)

		// API endpoints with auth
		http.HandleFunc("/api/commands", authMiddleware(apiCommandsHandler, token))
		http.HandleFunc("/api/delete", authMiddleware(apiDeleteHandler, token))
		http.HandleFunc("/api/clear", authMiddleware(apiClearHandler, token))
		http.HandleFunc("/api/stats", authMiddleware(apiStatsHandler, token))
		http.HandleFunc("/api/token", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"token": token})
		})

		// Serve HTML page
		http.HandleFunc("/", indexHandler)

		fmt.Printf("CMDO Server running at http://localhost:%s\n", port)
		fmt.Printf(" Using database: %s\n", dbPath)
		fmt.Printf(" Session token: %s\n", token)
		fmt.Println(" Auto-cleanup: Keeps last 1000 commands or 30 days (whichever is more)")
		fmt.Println(" Security: Token-based auth + localhost-only CORS")
		fmt.Println("Press Ctrl+C to stop")
		log.Fatal(http.ListenAndServe("127.0.0.1:"+port, nil))
	},
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Read embedded file
	content, err := indexHTML.ReadFile("server/index.html")
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		// log.Println("Error reading template:", err)
		return
	}

	// Parse template from embedded content
	tmpl, err := template.New("index").Parse(string(content))
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		// log.Println("Error parsing template:", err)
		return
	}

	tmpl.Execute(w, nil)
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func authMiddleware(next http.HandlerFunc, validToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS - only localhost
		origin := r.Header.Get("Origin")
		if origin != "" && (strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Auth-Token")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}

		// Check token
		token := r.Header.Get("X-Auth-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token != validToken {
			http.Error(w, "Unauthorized", 401)
			return
		}

		next(w, r)
	}
}

func apiCommandsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := database.DB.Query(`
		SELECT id, command, exit_code, timestamp, directory 
		FROM commands 
		ORDER BY timestamp DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var commands []CommandJSON
	for rows.Next() {
		var cmd CommandJSON

		// DIRECTLY STRING ME SCAN KARO
		err := rows.Scan(&cmd.ID, &cmd.Command, &cmd.ExitCode, &cmd.Timestamp, &cmd.Folder)
		if err != nil {
			continue
		}

		commands = append(commands, cmd)
	}

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

func apiStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var stats StatsJSON

	// Get total count
	err := database.DB.QueryRow("SELECT COUNT(*) FROM commands").Scan(&stats.TotalCommands)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get oldest and newest timestamps
	var oldest, newest string
	err = database.DB.QueryRow(`
		SELECT 
			MIN(timestamp) as oldest,
			MAX(timestamp) as newest
		FROM commands
	`).Scan(&oldest, &newest)

	if err != nil {
		stats.OldestTimestamp = "N/A"
		stats.NewestTimestamp = "N/A"
	} else {
		stats.OldestTimestamp = oldest
		stats.NewestTimestamp = newest
	}

	// Get database file size
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".cmdo", "cmdo.db")
	fileInfo, err := os.Stat(dbPath)
	if err == nil {
		sizeKB := float64(fileInfo.Size()) / 1024.0
		if sizeKB < 1024 {
			stats.DatabaseSize = fmt.Sprintf("%.2f KB", sizeKB)
		} else {
			stats.DatabaseSize = fmt.Sprintf("%.2f MB", sizeKB/1024.0)
		}
	} else {
		stats.DatabaseSize = "Unknown"
	}

	json.NewEncoder(w).Encode(stats)
}

func init() {
	serveCmd.Flags().String("port", "8089", "Port to run the server on")
	rootCmd.AddCommand(serveCmd)
}
