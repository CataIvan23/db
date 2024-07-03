package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq" // Driver PostgreSQL
)

// Structura pentru informatii despre sistemul de operare
type OSInfo struct {
	Nume           string `json:"nume"`
	Versiune       string `json:"versiune"`
	Arhitectura    string `json:"arhitectura"`
	DataInstalarii string `json:"data_instalarii"`
	Licenta        string `json:"licenta"`
}

// Structura pentru informatii despre hardware
type HardwareInfo struct {
	Procesor      string `json:"procesor"`
	Nuclee        int    `json:"nuclee"`
	FireExecutie  int    `json:"fire_executie"`
	Frecventa     string `json:"frecventa"`
	MemorieRAM    string `json:"memorie_ram"`
	TipStocare    string `json:"tip_stocare"`
	CapacitateHDD string `json:"capacitate_hdd"`
	PlacaDeBaza   string `json:"placa_de_baza"`
	PlacaVideo    string `json:"placa_video"`
}

// Structura pentru informatii despre software (programe instalate)
type SoftwareInfo struct {
	ProgrameInstalate []ProgramInfo `json:"programe_instalate"`
}

// Structura pentru informatii despre un program instalat
type ProgramInfo struct {
	Nume          string `json:"nume"`
	Versiune      string `json:"versiune"`
	Producator    string `json:"producator"`
	DataInstalare string `json:"data_instalare"`
	Licenta       string `json:"licenta"`
}

// Structura pentru informatii despre utilizator
type UserInfo struct {
	NumeUtilizator string `json:"nume_utilizator"`
	GrupUtilizator string `json:"grup_utilizator"`
}

// Structura pentru informatii live despre sistem
type LiveSystemInfo struct {
	UtilizareCPU      float64 `json:"utilizare_cpu"`
	UtilizareRAM      float64 `json:"utilizare_ram"`
	TraficTrimis      uint64  `json:"trafic_retea_bytes_trimisi"`
	TraficReceptionat uint64  `json:"trafic_retea_bytes_primiti"`
}

// Structura pentru a reprezenta întregul JSON
type SystemData struct {
	SistemDeOperare   *OSInfo       `json:"sistem_de_operare"`
	Hardware          *HardwareInfo `json:"hardware"`
	Software          *SoftwareInfo `json:"software"`
	Securitate        string        `json:"securitate"`
	Utilizator        *UserInfo     `json:"utilizator"`
	UtilizareCPU      float64       `json:"utilizare_cpu"`
	UtilizareRAM      float64       `json:"utilizare_ram"`
	TraficTrimis      uint64        `json:"trafic_retea_bytes_trimisi"`
	TraficReceptionat uint64        `json:"trafic_retea_bytes_primiti"`
}

// Funcție pentru a actualiza baza de date
func updateDatabase(db *sql.DB, data SystemData, idStatie int) error {
	_, err := db.Exec(`
		INSERT INTO statii_de_lucru (nume_statie) 
		VALUES ($1)
		ON CONFLICT (nume_statie) DO NOTHING;
	`, data.Utilizator.NumeUtilizator)
	if err != nil {
		return fmt.Errorf("eroare la actualizarea/inserarea în tabelul 'statii_de_lucru': %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO metadate_statii (
			id_statie, producator_procesor, model_procesor, nuclee, 
			fire_executie, frecventa, memorie_ram, tip_stocare, 
			capacitate_stocare, placa_de_baza, placa_video, 
			sistem_operare, versiune_software, arhitectura_sistem_operare, 
			data_instalare_sistem_operare, licenta_sistem_operare, securitate
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17
		)
		ON CONFLICT (id_statie) DO UPDATE SET 
			producator_procesor = EXCLUDED.producator_procesor,
			model_procesor = EXCLUDED.model_procesor,
			nuclee = EXCLUDED.nuclee,
			fire_executie = EXCLUDED.fire_executie,
			frecventa = EXCLUDED.frecventa,
			memorie_ram = EXCLUDED.memorie_ram,
			tip_stocare = EXCLUDED.tip_stocare,
			capacitate_stocare = EXCLUDED.capacitate_stocare,
			placa_de_baza = EXCLUDED.placa_de_baza,
			placa_video = EXCLUDED.placa_video,
			sistem_operare = EXCLUDED.sistem_operare,
			versiune_software = EXCLUDED.versiune_software,
			arhitectura_sistem_operare = EXCLUDED.arhitectura_sistem_operare,
			data_instalare_sistem_operare = EXCLUDED.data_instalare_sistem_operare,
			licenta_sistem_operare = EXCLUDED.licenta_sistem_operare,
			securitate = EXCLUDED.securitate
	`, idStatie, strings.Split(data.Hardware.PlacaDeBaza, " ")[0], data.Hardware.Procesor, data.Hardware.Nuclee,
		data.Hardware.FireExecutie, data.Hardware.Frecventa, data.Hardware.MemorieRAM, data.Hardware.TipStocare,
		data.Hardware.CapacitateHDD, data.Hardware.PlacaDeBaza, data.Hardware.PlacaVideo,
		data.SistemDeOperare.Nume, data.SistemDeOperare.Versiune, data.SistemDeOperare.Arhitectura,
		data.SistemDeOperare.DataInstalarii, data.SistemDeOperare.Licenta, data.Securitate)
	if err != nil {
		return fmt.Errorf("eroare la actualizarea metadatelor stației: %w", err)
	}

	for _, program := range data.Software.ProgrameInstalate {
		_, err = db.Exec(`
			INSERT INTO software_instalat (id_statie, nume, versiune, producator, data_instalare, licenta)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id_statie, nume, versiune) DO NOTHING
		`, idStatie, program.Nume, program.Versiune, program.Producator, program.DataInstalare, program.Licenta)
		if err != nil {
			return fmt.Errorf("eroare la actualizarea software-ului instalat: %w", err)
		}
	}

	_, err = db.Exec(`
		INSERT INTO metrici_statii (id_statie, timestamp, utilizare_cpu, utilizare_memorie, trafic_retea_bytes_trimisi, trafic_retea_bytes_primiti) 
		VALUES ($1, NOW(), $2, $3, $4, $5)
	`, idStatie, data.UtilizareCPU, data.UtilizareRAM, data.TraficTrimis, data.TraficReceptionat)
	if err != nil {
		return fmt.Errorf("eroare la inserarea metricii stației: %w", err)
	}

	return nil
}

func handleJSONData(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Eroare la citirea corpului cererii", http.StatusBadRequest)
			return
		}

		var jsonData SystemData
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			http.Error(w, "Eroare la parsarea JSON", http.StatusBadRequest)
			return
		}

		numeStatie := jsonData.Utilizator.NumeUtilizator
		var idStatie int
		err = db.QueryRow("SELECT id_statie FROM statii_de_lucru WHERE nume_statie = $1", numeStatie).Scan(&idStatie)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Stația de lucru nu a fost găsită în baza de date", http.StatusNotFound)
				return
			} else {
				http.Error(w, fmt.Sprintf("Eroare la interogarea bazei de date: %v", err), http.StatusInternalServerError)
				return
			}
		}

		err = updateDatabase(db, jsonData, idStatie)
		if err != nil {
			http.Error(w, fmt.Sprintf("Eroare la actualizarea bazei de date: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Datele JSON au fost primite și baza de date a fost actualizată cu succes!")
	} else {
		http.Error(w, "Metodă HTTP neacceptată", http.StatusMethodNotAllowed)
	}
}

func main() {
	connStr := "postgres://postgres:password@localhost/postgres?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Eroare la conectarea la baza de date: %v\n", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Eroare la verificarea conexiunii la baza de date: %v\n", err)
	}

	go func() {
		http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
			handleJSONData(w, r, db)
		})

		log.Printf("Serverul HTTP pornit și ascultă pe portul :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	for {
		data, err := os.ReadFile("live_data.json")
		if err != nil {
			log.Fatalf("Eroare la citirea fișierului JSON: %v\n", err)
		}

		req, err := http.NewRequest("POST", "http://localhost:8080/data", bytes.NewBuffer(data))
		if err != nil {
			log.Fatalf("Eroare la crearea cererii: %v\n", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Eroare la trimiterea cererii: %v\n", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Eroare la citirea răspunsului: %v\n", err)
		}

		fmt.Println("Răspuns de la server:", string(body))

		time.Sleep(10 * time.Second)
	}
}
