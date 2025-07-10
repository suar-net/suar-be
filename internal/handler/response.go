package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

// respondWithError adalah helper untuk mengirim respons error dalam format JSON.
// Ini adalah cara standar untuk melaporkan error di API.
func respondWithError(w http.ResponseWriter, code int, message string) {
	// Untuk konsistensi, kita bungkus pesan error dalam sebuah objek JSON.
	// Contoh: {"error": "Pesan errornya apa"}
	respondWithJson(w, code, map[string]string{"error": message})
}

// respondWithJson adalah helper serbaguna untuk mengirim respons dalam format JSON.
// Fungsi ini menangani marshaling, setting header, dan penulisan respons.
// 'payload' menggunakan 'interface{}' agar bisa menerima tipe data apa pun (struct, map, dll).
func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	// 1. Marshal payload ke JSON
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v", payload)
		// Jika marshaling gagal, ini adalah kesalahan server
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 2. Set header Content-Type
	w.Header().Set("Content-Type", "application/json")

	// 3. Tulis status code HTTP
	w.WriteHeader(code)

	// 4. Tulis body JSON
	w.Write(dat)
}
