package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "modernc.org/sqlite"
)

type Application struct {
	database *sql.DB
}

type Todo struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func main() {
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal(err)
	}

	database, err := sql.Open(
		"sqlite",
		"file:data/todo.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
	)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	database.SetMaxOpenConns(1)

	if err := createSchema(database); err != nil {
		log.Fatal(err)
	}

	application := &Application{
		database: database,
	}

	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/api/todos", func(router chi.Router) {
		router.Get("/", application.listTodos)
		router.Post("/", application.createTodo)

		router.Route("/{id}", func(router chi.Router) {
			router.Patch("/", application.updateTodo)
			router.Delete("/", application.deleteTodo)
		})
	})

	// PrimalJS + application frontend
	fileServer := http.FileServer(http.Dir("./web"))
	router.Handle("/*", fileServer)

	log.Println("Todo app running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}

func createSchema(database *sql.DB) error {
	_, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			completed INTEGER NOT NULL DEFAULT 0
		)
	`)

	return err
}

func (application *Application) listTodos(
	response http.ResponseWriter,
	request *http.Request,
) {
	rows, err := application.database.Query(`
		SELECT id, title, completed
		FROM todos
		ORDER BY id
	`)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	todos := make([]Todo, 0)

	for rows.Next() {
		var todo Todo
		var completed int

		if err := rows.Scan(
			&todo.ID,
			&todo.Title,
			&completed,
		); err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}

		todo.Completed = completed != 0
		todos = append(todos, todo)
	}

	writeJSON(response, http.StatusOK, todos)
}

func (application *Application) createTodo(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(response, "Invalid request", http.StatusBadRequest)
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		http.Error(response, "Title is required", http.StatusBadRequest)
		return
	}
	result, err := application.database.Exec(
		`INSERT INTO todos (title, completed) VALUES (?, 0)`,
		input.Title,
	)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	todo := Todo{
		ID:        id,
		Title:     input.Title,
		Completed: false,
	}
	writeJSON(response, http.StatusCreated, todo)
}

func (application *Application) updateTodo(
	response http.ResponseWriter,
	request *http.Request,
) {
	id := chi.URLParam(request, "id")
	var input struct {
		Completed *bool `json:"completed"`
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(response, "Invalid request", http.StatusBadRequest)
		return
	}
	if input.Completed == nil {
		http.Error(response, "completed is required", http.StatusBadRequest)
		return
	}
	completed := 0
	if *input.Completed {
		completed = 1
	}
	result, err := application.database.Exec(
		`UPDATE todos SET completed = ? WHERE id = ?`,
		completed,
		id,
	)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(response, "Todo not found", http.StatusNotFound)
		return
	}
	var todo Todo
	var storedCompleted int
	err = application.database.QueryRow(`
		SELECT id, title, completed
		FROM todos
		WHERE id = ?
	`, id).Scan(
		&todo.ID,
		&todo.Title,
		&storedCompleted,
	)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	todo.Completed = storedCompleted != 0
	writeJSON(response, http.StatusOK, todo)
}

func (application *Application) deleteTodo(
	response http.ResponseWriter,
	request *http.Request,
) {
	id := chi.URLParam(request, "id")
	result, err := application.database.Exec(
		`DELETE FROM todos WHERE id = ?`,
		id,
	)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(response, "Todo not found", http.StatusNotFound)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func writeJSON(
	response http.ResponseWriter,
	status int,
	value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(response).Encode(value)
	}
}