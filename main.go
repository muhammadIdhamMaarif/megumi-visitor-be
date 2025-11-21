package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Visitor struct {
	Nama         string `json:"nama"`
	Instansi     string `json:"instansi"`
	Kontak       string `json:"kontak"`
	PicLab       string `json:"picLab"`
	Tujuan       string `json:"tujuan"`
	TujuanCustom string `json:"tujuanCustom"`
	CreatedAt    string `json:"createdAt,omitempty"`
}

var db *sql.DB

func main() {
	// Example DSN:
	//   user:password@tcp(127.0.0.1:3306)/lab_mgm?parseTime=true
	// Set this in GoLand Run Configuration or your shell.
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("DATABASE_DSN environment variable is not set")
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("error opening DB: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("error pinging DB: %v", err)
	}
	log.Println("Connected to MySQL successfully")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/visitors", visitorsHandler)

	// Wrap with CORS middleware
	handler := withCORS(mux)

	addr := ":8080"
	log.Printf("Server running on %s ...", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// CORS middleware: adjust origin to your frontend URL
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Change this to your actual frontend origin if needed
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func visitorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		handleCreateVisitor(w, r)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func handleCreateVisitor(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var v Visitor
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if v.Nama == "" || v.Instansi == "" || v.Kontak == "" || v.PicLab == "" || v.Tujuan == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	if v.Tujuan == "lainnya" && v.TujuanCustom == "" {
		http.Error(w, "tujuanCustom is required when tujuan is 'lainnya'", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO visitors (nama, instansi, kontak, pic_lab, tujuan, tujuan_custom, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()

	_, err := db.Exec(
		query,
		v.Nama,
		v.Instansi,
		v.Kontak,
		v.PicLab,
		v.Tujuan,
		nullIfEmpty(v.TujuanCustom),
		now,
	)
	if err != nil {
		log.Printf("error inserting visitor: %v", err)
		http.Error(w, "failed to save data", http.StatusInternalServerError)
		return
	}

	v.CreatedAt = now.Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "visitor saved",
		"data":    v,
	})
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
