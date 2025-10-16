package main

import (

	// "text/template"

	// "log"
	// "net/http"

	"github.com/tanu2534/cmdo/cmd"
	// "github.com/tanu2534/cmdo/database"
	// "os"
)

func main() {

	// fmt.Println("Welcome to CMDO v0.1.0")
	cmd.Execute()
	// // database.InitDB("./cmdo.db")
	// // defer database.DB.Close()
	// http.HandleFunc("/", commandsHandler)
	// log.Fatal(http.ListenAndServe(":8089", nil))
}
