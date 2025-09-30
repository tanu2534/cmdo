package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

type Command struct {
	ID        int
	Command   string
	Directory string
	ExitCode  string
	Timestamp string
}

func DeleteCommand(id string) error {
	if DB == nil {
		log.Println("DB is not initialized")
		return sql.ErrConnDone
	}

	_, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	_, err = DB.Exec("DELETE FROM commands WHERE id = ?", id)
	if err != nil {
		log.Printf("Error deleting command with id %s: %v", id, err)
		return err
	}
	return nil
}

func GetCommandsGrouped() (map[string][]Command, error) {
	// DB check karo
	if DB == nil {
		log.Println("DB is nil, reopening...")
		InitDB("./cmdo.db") // dbPath ko global variable me store karna hoga InitDB call ke time
	}

	// Agar DB closed hai to bhi handle karo
	if err := DB.Ping(); err != nil {
		log.Println("DB was closed, reopening...")
		InitDB("./cmdo.db")
	}

	rows, err := DB.Query(`SELECT id, command, directory, exit_code, timestamp 
		FROM commands ORDER BY directory, timestamp DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grouped := make(map[string][]Command)

	for rows.Next() {
		var c Command
		if err := rows.Scan(&c.ID, &c.Command, &c.Directory, &c.ExitCode, &c.Timestamp); err != nil {
			log.Println("Error scanning row:", err)
			continue
		}
		grouped[c.Directory] = append(grouped[c.Directory], c)
	}

	fmt.Println(grouped)
	return grouped, nil
}

func InitDB(path string) {
	var err error
	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS commands  (
	       id INTEGER PRIMARY KEY,
	       command TEXT,
	       directory TEXT,
	       exit_code INTEGER,
	       timestamp TEXT
	);`

	_, err = DB.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("Error creating table: %q: %s\n", err, sqlStmt)
	}

	fmt.Println("DB Initialized")
}

func InsertCmd(cmd string, code string, dir string) {
	if DB == nil {
		log.Fatal("DB is not initialized. Call InitDB first.")
	}

	sqlStmt := `INSERT INTO commands(command, exit_code, directory, timestamp) VALUES(?, ?, ?, ?)`
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	_, err := DB.Exec(sqlStmt, cmd, code, dir, timestamp)
	if err != nil {
		log.Printf("Error inserting data: %s\n", err)
	}
}
