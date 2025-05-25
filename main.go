package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/adehusnim37/lihatin-go/routes"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/go-playground/validator/v10"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "root:rootpassword123@tcp(localhost:3306)/LihatinGo")
	if err != nil {
		panic(err)
		log.Printf("Please check your database connection during hitting the server")
		log.Printf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Initialize validator with custom rules
	validate := validator.New()
	utils.SetupCustomValidators(validate)

	r := routes.SetupRouter(db, validate)
	fmt.Println("Server running on localhost:8880 Please check your browser")
	r.Run(":8880")
}
