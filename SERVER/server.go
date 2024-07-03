package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Metodă nepermisă", http.StatusMethodNotAllowed)
			return
		}

		// Citește corpul cererii (datele JSON)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Eroare la citirea corpului cererii", http.StatusInternalServerError)
			return
		}

		// Salvează datele JSON într-un fișier
		fileName := "received_data.json"
		err = os.WriteFile(fileName, body, 0644)
		if err != nil {
			http.Error(w, "Eroare la salvarea datelor JSON", http.StatusInternalServerError)
			return
		}

		// Răspunde clientului cu un mesaj de confirmare
		fmt.Fprintf(w, "Datele JSON au fost primite cu succes și salvate în %s!", fileName)
	})

	fmt.Printf("Serverul ascultă pe portul 8080...\n")
	http.ListenAndServe(":8080", nil)
}
