package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq" // Driver PostgreSQL
)

// Structura pentru informații despre sistemul de operare
type OSInfo struct {
	Nume           string `json:"nume"`
	Versiune       string `json:"versiune"`
	Arhitectura    string `json:"arhitectura"`
	DataInstalarii string `json:"data_instalarii"`
	Licenta        string `json:"licenta"`
}

// Structura pentru informații despre hardware
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

// Structura pentru informații despre software (programe instalate)
type SoftwareInfo struct {
	ProgrameInstalate []ProgramInfo `json:"programe_instalate"`
}

// Structura pentru informații despre un program instalat
type ProgramInfo struct {
	Nume          string `json:"nume"`
	Versiune      string `json:"versiune"`
	Producator    string `json:"producator"`
	DataInstalare string `json:"data_instalare"`
	Licenta       string `json:"licenta"`
	// Alte informații despre program
}

// Structura pentru informații live despre sistem
type LiveSystemInfo struct {
	UtilizareCPU      float64 `json:"utilizare_cpu"`
	UtilizareRAM      float64 `json:"utilizare_ram"`
	TraficTrimis      uint64  `json:"trafic_retea_bytes_trimisi"`
	TraficReceptionat uint64  `json:"trafic_retea_bytes_primiti"`
}

// Funcție pentru a încărca datele JSON din fișier
func loadDataFromFile(fileName string) (map[string]interface{}, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("eroare la citirea fișierului JSON: %w", err)
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return nil, fmt.Errorf("eroare la parsarea JSON: %w", err)
	}

	return jsonData, nil
}

// Funcție pentru a obține ID-ul stației curente din baza de date
// sau pentru a crea o intrare nouă dacă nu există
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

func updateDatabase(db *sql.DB, systemInfo map[string]interface{}, idStatie int) error {
	// Verifică și extrage informațiile necesare din `systemInfo`
	osInfo, ok := systemInfo["sistem_de_operare"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("cheia 'sistem_de_operare' lipsește sau nu este de tipul map[string]interface{}")
	}

	hardwareInfo, ok := systemInfo["hardware"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("cheia 'hardware' lipsește sau nu este de tipul map[string]interface{}")
	}

	softwareInfo, ok := systemInfo["software"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("cheia 'software' lipsește sau nu este de tipul map[string]interface{}")
	}

	securityInfo, ok := systemInfo["securitate"].(string)
	if !ok {
		return fmt.Errorf("cheia 'securitate' lipsește sau nu este de tipul string")
	}

	// Extrage informațiile live despre sistem doar dacă există
	liveInfo, ok := systemInfo["live_info"].(map[string]interface{})
	if ok {
		// Verificăm cheile din `liveInfo` doar dacă `liveInfo` există
		utilizareCPU, ok := liveInfo["utilizare_cpu"].(float64)
		if !ok {
			return fmt.Errorf("cheia 'utilizare_cpu' lipsește sau nu este de tipul float64 în 'live_info'")
		}

		utilizareRAM, ok := liveInfo["utilizare_ram"].(float64)
		if !ok {
			return fmt.Errorf("cheia 'utilizare_ram' lipsește sau nu este de tipul float64 în 'live_info'")
		}

		traficTrimis, ok := liveInfo["trafic_retea_bytes_trimisi"].(float64)
		if !ok {
			return fmt.Errorf("cheia 'trafic_retea_bytes_trimisi' lipsește sau nu este de tipul float64 în 'live_info'")
		}

		traficReceptionat, ok := liveInfo["trafic_retea_bytes_primiti"].(float64)
		if !ok {
			return fmt.Errorf("cheia 'trafic_retea_bytes_primiti' lipsește sau nu este de tipul float64 în 'live_info'")
		}

		// 5. Inserare în tabel 'metrici_statii' doar dacă 'live_info' există
		_, err := db.Exec(`
			INSERT INTO metrici_statii (id_statie, timestamp, utilizare_cpu, utilizare_memorie, trafic_retea_bytes_trimisi, trafic_retea_bytes_primiti) 
			VALUES ($1, NOW(), $2, $3, $4, $5)
		`, idStatie, utilizareCPU, utilizareRAM, traficTrimis, traficReceptionat)
		if err != nil {
			return fmt.Errorf("eroare la inserarea metricii stației: %w", err)
		}
	}

	// Actualizare sau inserare în tabel 'metadate_statii'
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
	`, idStatie, strings.Split(hardwareInfo["placa_de_baza"].(string), " ")[0], hardwareInfo["procesor"], hardwareInfo["nuclee"],
		hardwareInfo["fire_executie"], hardwareInfo["frecventa"], hardwareInfo["memorie_ram"], hardwareInfo["tip_stocare"],
		hardwareInfo["capacitate_hdd"], hardwareInfo["placa_de_baza"], hardwareInfo["placa_video"],
		osInfo["nume"], osInfo["versiune"], osInfo["arhitectura"],
		osInfo["data_instalarii"], osInfo["licenta"], securityInfo)
	if err != nil {
		return fmt.Errorf("eroare la actualizarea metadatelor stației: %w", err)
	}

	// 4. Actualizare tabel 'software_instalat'
	for _, program := range softwareInfo["programe_instalate"].([]interface{}) {
		programMap, ok := program.(map[string]interface{})
		if !ok {
			return fmt.Errorf("elementul din 'programe_instalate' nu este de tipul map[string]interface{}")
		}
		_, err = db.Exec(`
			INSERT INTO software_instalat (id_statie, nume, versiune, producator, data_instalare, licenta)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id_statie, nume, versiune) DO NOTHING
		`, idStatie, programMap["nume"], programMap["versiune"], programMap["producator"], programMap["data_instalare"], programMap["licenta"])
		if err != nil {
			return fmt.Errorf("eroare la actualizarea software-ului instalat: %w", err)
		}
	}

	return nil
}

func main() {
	// Informații despre conexiunea la baza de date
	connStr := "postgres://postgres:password@localhost/postgres?sslmode=disable"

	// Conectare la baza de date
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Eroare la conectarea la baza de date: %v\n", err)
		return
	}
	defer db.Close()

	// Verificare conexiune
	err = db.Ping()
	if err != nil {
		fmt.Printf("Eroare la verificarea conexiunii la baza de date: %v\n", err)
		return
	}

	// *Obțineți ID-ul stației curente (dacă există) sau creați o intrare nouă*
	numeStatie := "username" // Înlocuiți cu o metodă potrivită de identificare a stației
	idStatie, err := getStationID(db, numeStatie)
	if err != nil {
		fmt.Printf("Eroare la obținerea/crearea ID-ului stației: %v\n", err)
		return
	}

	fmt.Printf("ID-ul stației curente: %d\n", idStatie)

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

		// Încarcă datele JSON din fișier
		jsonData, err := loadDataFromFile(fileName)
		if err != nil {
			fmt.Printf("Eroare la încărcarea datelor din fișierul JSON: %v\n", err)
			http.Error(w, "Eroare la încărcarea datelor din fișierul JSON", http.StatusInternalServerError)
			return
		}

		// Actualizare baza de date
		err = updateDatabase(db, jsonData, idStatie)
		if err != nil {
			fmt.Printf("Eroare la actualizarea bazei de date: %v\n", err)
			http.Error(w, "Eroare la actualizarea bazei de date", http.StatusInternalServerError)
			return
		}

		fmt.Println("Baza de date actualizată cu succes!")
		// Răspunde clientului cu un mesaj de confirmare
		fmt.Fprintf(w, "Datele JSON au fost primite cu succes și salvate în %s!", fileName)
	})

	fmt.Printf("Serverul ascultă pe portul 8080...\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Eroare la pornirea serverului: %v\n", err)
	}
}
