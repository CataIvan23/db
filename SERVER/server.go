package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"database/sql"

	_ "github.com/lib/pq" // Driver PostgreSQL
)

// Structuri pentru datele primite de la client
type OSInfo struct {
	Nume           string `json:"nume"`
	Versiune       string `json:"versiune"`
	Arhitectura    string `json:"arhitectura"`
	DataInstalarii string `json:"data_instalarii"`
	Licenta        string `json:"licenta"`
}

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

type SoftwareInfo struct {
	ProgrameInstalate []ProgramInfo `json:"programe_instalate"`
}

type ProgramInfo struct {
	Nume          string `json:"nume"`
	Versiune      string `json:"versiune"`
	Producator    string `json:"producator"`
	DataInstalare string `json:"data_instalare"`
	Licenta       string `json:"licenta"`
}

type UserInfo struct {
	NumeUtilizator string `json:"nume_utilizator"`
	GrupUtilizator string `json:"grup_utilizator"`
}

type LiveSystemInfo struct {
	UtilizareCPU      float64 `json:"utilizare_cpu"`
	UtilizareRAM      float64 `json:"utilizare_ram"`
	TraficTrimis      uint64  `json:"trafic_retea_bytes_trimisi"`
	TraficReceptionat uint64  `json:"trafic_retea_bytes_primiti"`
}

type SystemInfo struct {
	SistemDeOperare   OSInfo       `json:"sistem_de_operare"`
	Hardware          HardwareInfo `json:"hardware"`
	Software          SoftwareInfo `json:"software"`
	Securitate        string       `json:"securitate"`
	Utilizator        UserInfo     `json:"utilizator"`
	UtilizareCPU      float64      `json:"utilizare_cpu"`
	UtilizareRAM      float64      `json:"utilizare_ram"`
	TraficTrimis      uint64       `json:"trafic_retea_bytes_trimisi"`
	TraficReceptionat uint64       `json:"trafic_retea_bytes_primiti"`
}

func updateDatabase(db *sql.DB, systemInfo *SystemInfo, idStatie int) error {
	// 3. Actualizare sau inserare în tabel 'metadate_statii'
	_, err := db.Exec(`
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
	`, idStatie, strings.Split(systemInfo.Hardware.PlacaDeBaza, " ")[0], systemInfo.Hardware.Procesor, systemInfo.Hardware.Nuclee,
		systemInfo.Hardware.FireExecutie, systemInfo.Hardware.Frecventa, systemInfo.Hardware.MemorieRAM, systemInfo.Hardware.TipStocare,
		systemInfo.Hardware.CapacitateHDD, systemInfo.Hardware.PlacaDeBaza, systemInfo.Hardware.PlacaVideo,
		systemInfo.SistemDeOperare.Nume, systemInfo.SistemDeOperare.Versiune, systemInfo.SistemDeOperare.Arhitectura,
		systemInfo.SistemDeOperare.DataInstalarii, systemInfo.SistemDeOperare.Licenta, systemInfo.Securitate)
	if err != nil {
		return fmt.Errorf("eroare la actualizarea metadatelor stației: %w", err)
	}

	// 4. Actualizare tabel 'software_instalat'
	for _, program := range systemInfo.Software.ProgrameInstalate {
		_, err = db.Exec(`
			INSERT INTO software_instalat (id_statie, nume, versiune, producator, data_instalare, licenta)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id_statie, nume, versiune) DO NOTHING
		`, idStatie, program.Nume, program.Versiune, program.Producator, program.DataInstalare, program.Licenta)
		if err != nil {
			return fmt.Errorf("eroare la actualizarea software-ului instalat: %w", err)
		}
	}

	// 5. Inserare în tabel 'metrici_statii'
	_, err = db.Exec(`
		INSERT INTO metrici_statii (id_statie, timestamp, utilizare_cpu, utilizare_memorie, trafic_retea_bytes_trimisi, trafic_retea_bytes_primiti) 
		VALUES ($1, NOW(), $2, $3, $4, $5)
	`, idStatie, systemInfo.UtilizareCPU, systemInfo.UtilizareRAM, systemInfo.TraficTrimis, systemInfo.TraficReceptionat)
	if err != nil {
		return fmt.Errorf("eroare la inserarea metricii stației: %w", err)
	}

	return nil
}

func getStationID(db *sql.DB, numeStatie string) (int, error) {
	var idStatie int
	err := db.QueryRow("SELECT id_statie FROM statii_de_lucru WHERE nume_statie = $1", numeStatie).Scan(&idStatie)
	if err != nil {
		if err == sql.ErrNoRows {
			// Nu există o intrare pentru această stație, deci o creăm
			// Trebuie să furnizați informațiile necesare pentru tabelul 'statii_de_lucru'
			// Aici presupunem că există un utilizator cu ID-ul 1
			err = db.QueryRow("INSERT INTO statii_de_lucru (nume_statie, id_persoana) VALUES ($1, 1) RETURNING id_statie", numeStatie).Scan(&idStatie)
			if err != nil {
				return 0, fmt.Errorf("eroare la crearea intrării stației de lucru: %w", err)
			}
			return idStatie, nil
		} else {
			return 0, fmt.Errorf("eroare la interogarea bazei de date: %w", err)
		}
	}
	return idStatie, nil
}

func handleUpdate(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metoda nepermisă", http.StatusMethodNotAllowed)
		return
	}

	// Citește datele JSON din corpul cererii
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Eroare la citirea datelor din cerere", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Deserializează datele JSON
	var systemInfo SystemInfo
	err = json.Unmarshal(body, &systemInfo)
	if err != nil {
		http.Error(w, "Eroare la parsarea JSON", http.StatusBadRequest)
		return
	}

	numeStatie := systemInfo.Utilizator.NumeUtilizator
	idStatie, err := getStationID(db, numeStatie)
	if err != nil {
		fmt.Printf("Eroare la obținerea/crearea ID-ului stației: %v\n", err)
		return
	}

	fmt.Printf("ID-ul stației curente: %d\n", idStatie)

	// Actualizare baza de date
	err = updateDatabase(db, &systemInfo, idStatie)
	if err != nil {
		http.Error(w, "Eroare la actualizarea bazei de date", http.StatusInternalServerError)
		return
	}

	// Răspuns de succes
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Datele au fost actualizate cu succes în baza de date!")
}

func main() {
	// Informații despre conexiunea la baza de date
	connStr := "postgres://postgres:password@localhost/postgres?sslmode=disable"

	// Conectare la baza de date
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Eroare la conectarea la baza de date: %v", err)
	}
	defer db.Close()

	// Verificare conexiune
	err = db.Ping()
	if err != nil {
		log.Fatalf("Eroare la verificarea conexiunii la baza de date: %v", err)
	}

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		handleUpdate(w, r, db)
	})

	fmt.Println("Serverul ascultă pe portul 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
