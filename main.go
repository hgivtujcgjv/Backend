package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql" // Импортируем драйвер MySQL
	gorilla "github.com/gorilla/mux"
)

func main() {
	dsn := "root:123@tcp(127.0.0.1:3306)/hw9up?charset=utf8&interpolateParams=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("cant open db, err: %v\n", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("cant connect to db, err: %v\n", err)
	}
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		log.Fatalf("error fetching tables, err: %v\n", err)
	}
	defer rows.Close()

	fmt.Println("Tables in the database:")
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Fatalf("error scanning table name, err: %v\n", err)
		}
		fmt.Println(tableName)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("error during rows iteration, err: %v\n", err)
	}

	sm := NewSessionsDB(db)
	u := &UserHandler{
		Bd:   db,
		Sess: sm,
	}
	mux := gorilla.NewRouter()
	mux.HandleFunc("/users/login", u.Login)
	mux.HandleFunc("/articles", u.SwitchArticlesMethods) // Обрабатывает ?
	mux.HandleFunc("/articles/{anything:.*}", u.GetArticleMassive)
	mux.HandleFunc("/users", u.SwitchUserMethods)
	http.Handle("/", AuthMiddleware(sm, mux))
	addr := ":8080"
	h := GetApp()
	fmt.Println("start server at", addr)
	http.ListenAndServe(addr, h)
	log.Fatalf("Server failed: %v\n", err)
}
