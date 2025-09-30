package main

import (
	"fmt"
	// "text/template"

	// "log"
	// "net/http"

	"github.com/tanu2534/cmdo/cmd"
	// "github.com/tanu2534/cmdo/database"
	// "os"
)

func main() {

	fmt.Println("ddddd")
	cmd.Execute()
	// // database.InitDB("./cmdo.db")
	// // defer database.DB.Close()
	// http.HandleFunc("/", commandsHandler)
	// log.Fatal(http.ListenAndServe(":8089", nil))
}
