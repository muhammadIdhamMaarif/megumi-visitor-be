package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type Handler struct {
	db *sql.DB
}

// Request payloads
type VisitorRequest struct {
	Nama         string `json:"nama"`
	Instansi     string `json:"instansi"`
	Kontak       string `json:"kontak"`
	PICLab       string `json:"pic_lab"`
	Tujuan       string `json:"tujuan"`
	TujuanCustom string `json:"tujuan_custom"`
}

type UserRequest struct {
	Nama   string `json:"nama"`
	NIM    string `json:"nim"`
	Kontak string `json:"kontak"`
}

type ManagerRequest struct {
	Nama string `json:"nama"`
}

// Generic response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	// Load .env (ignore error if file not present, env may come from Docker, etc.)
	_ = godotenv.Load()

	db, err := initDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	handler := &Handler{db: db}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/visitor-form", handler.CreateVisitor)
	mux.HandleFunc("/api/v1/user-form", handler.CreateUser)
	mux.HandleFunc("/api/v1/manager-form", handler.CreateManager)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("Server is running on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func initDB() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")

	if host == "" || port == "" || user == "" || name == "" {
		return nil, fmt.Errorf("database configuration is incomplete; check .env")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		user, pass, host, port, name)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Optional tuning
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// Helpers

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, APIResponse{
		Success: false,
		Message: "method not allowed",
	})
}

// Handlers

// POST /api/v1/visitor-form
func (h *Handler) CreateVisitor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req VisitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "invalid JSON body",
		})
		return
	}

	// Simple validation (you can expand this as needed)
	if req.Nama == "" || req.Instansi == "" || req.Kontak == "" || req.PICLab == "" || req.Tujuan == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "missing required fields",
		})
		return
	}

	var tujuanCustom interface{}
	if req.TujuanCustom == "" {
		tujuanCustom = nil // will be stored as NULL
	} else {
		tujuanCustom = req.TujuanCustom
	}

	res, err := h.db.Exec(
		`INSERT INTO visitors (nama, instansi, kontak, pic_lab, tujuan, tujuan_custom)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		req.Nama, req.Instansi, req.Kontak, req.PICLab, req.Tujuan, tujuanCustom,
	)
	if err != nil {
		log.Printf("error inserting visitor: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "failed to save visitor data",
		})
		return
	}

	id, _ := res.LastInsertId()

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "visitor data saved",
		Data: map[string]interface{}{
			"id": id,
		},
	})
}

// POST /api/v1/user-form
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "invalid JSON body",
		})
		return
	}

	if req.Nama == "" || req.NIM == "" || req.Kontak == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "missing required fields",
		})
		return
	}

	res, err := h.db.Exec(
		`INSERT INTO users (nama, nim, kontak)
		 VALUES (?, ?, ?)`,
		req.Nama, req.NIM, req.Kontak,
	)
	if err != nil {
		log.Printf("error inserting user: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "failed to save user data",
		})
		return
	}

	id, _ := res.LastInsertId()

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "user data saved",
		Data: map[string]interface{}{
			"id": id,
		},
	})
}

// POST /api/v1/manager-form
func (h *Handler) CreateManager(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req ManagerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "invalid JSON body",
		})
		return
	}

	if req.Nama == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "missing required field: nama",
		})
		return
	}

	res, err := h.db.Exec(
		`INSERT INTO managers (nama)
		 VALUES (?)`,
		req.Nama,
	)
	if err != nil {
		log.Printf("error inserting manager: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "failed to save manager data",
		})
		return
	}

	id, _ := res.LastInsertId()

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "manager data saved",
		Data: map[string]interface{}{
			"id": id,
		},
	})
}
