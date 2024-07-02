package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"

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

// Functie pentru a obtine informatii despre sistemul de operare
func getOSInfo() (*OSInfo, error) {
	osInfo := &OSInfo{}

	// Obtine numele si versiunea sistemului de operare
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "ver")
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("eroare la executarea comenzii 'ver': %w", err)
		}
		osInfo.Nume = strings.TrimSpace(string(out))

		// Obtine arhitectura sistemului de operare
		osInfo.Arhitectura = runtime.GOARCH

		// Obtine data instalarii sistemului de operare (pentru Windows)
		cmd = exec.Command("cmd", "/c", "wmic os get InstallDate /VALUE")
		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("eroare la executarea comenzii 'wmic os get InstallDate': %w", err)
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "=")
		if len(lines) > 1 {
			dateString := strings.TrimSpace(lines[1])

			// Soluția 1: Ajustarea formatului datei
			// dataInstalare, err := time.Parse("20060102150405.000000-0700", dateString) // Formatul corect pentru fusul orar
			// if err != nil {
			// 	return nil, fmt.Errorf("eroare la parsarea datei de instalare: %w", err)
			// }

			// Soluția 2: Eliminarea fusului orar
			if strings.Contains(dateString, "+") {
				parts := strings.Split(dateString, "+")
				dateString = parts[0] // Păstrează doar partea de dată
			}

			// Parsare DataInstalarii în format time.Time (fără fus orar)
			dataInstalare, err := time.Parse("20060102150405.000000", dateString)
			if err != nil {
				return nil, fmt.Errorf("eroare la parsarea datei de instalare: %w", err)
			}

			// Formatare DataInstalarii în ISO 8601
			osInfo.DataInstalarii = dataInstalare.Format("2006-01-02 15:04:05")
		}

		// Obtine licenta sistemului de operare (daca este cazul)
		// Implementati aici logica specifica pentru licenta Windows
		osInfo.Licenta = "N/A"

	case "linux":
		// Implementati logica pentru Linux
		osInfo.Nume = "Linux" // Exemplu simplificat pentru numele sistemului de operare
		osInfo.Versiune = "N/A"
		osInfo.Arhitectura = runtime.GOARCH
		osInfo.DataInstalarii = "N/A"
		osInfo.Licenta = "N/A"

	case "darwin":
		// Implementati logica pentru macOS
		cmd := exec.Command("sw_vers", "-productName")
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("eroare la executarea comenzii 'sw_vers -productName': %w", err)
		}
		osInfo.Nume = strings.TrimSpace(string(out))

		cmd = exec.Command("sw_vers", "-productVersion")
		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("eroare la executarea comenzii 'sw_vers -productVersion': %w", err)
		}
		osInfo.Versiune = strings.TrimSpace(string(out))

		osInfo.Arhitectura = runtime.GOARCH
		osInfo.DataInstalarii = "N/A" // Exemplu simplificat pentru macOS
		osInfo.Licenta = "N/A"

	default:
		return nil, fmt.Errorf("sistem de operare neacceptat: %s", runtime.GOOS)
	}

	return osInfo, nil
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

// Functie pentru a obtine informatii despre hardware
func getHardwareInfo() (*HardwareInfo, error) {
	hardwareInfo := &HardwareInfo{}

	// Obtine informatii despre procesor
	cmd := exec.Command("cmd", "/c", "wmic cpu get Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed /FORMAT:LIST")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("eroare la executarea comenzii 'wmic cpu get ...': %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "=") {
			fields := strings.Split(line, "=")
			switch strings.TrimSpace(fields[0]) {
			case "Name":
				hardwareInfo.Procesor = strings.TrimSpace(fields[1])
			case "NumberOfCores":
				hardwareInfo.Nuclee, _ = strconv.Atoi(strings.TrimSpace(fields[1]))
			case "NumberOfLogicalProcessors":
				hardwareInfo.FireExecutie, _ = strconv.Atoi(strings.TrimSpace(fields[1]))
			case "MaxClockSpeed":
				hardwareInfo.Frecventa = strings.TrimSpace(fields[1]) + " MHz"
			}
		}
	}

	// Obtine informatii despre memoria RAM
	cmd = exec.Command("cmd", "/c", "wmic memorychip get Capacity /FORMAT:LIST")
	out, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("eroare la executarea comenzii 'wmic memorychip get Capacity': %w", err)
	}
	lines = strings.Split(strings.TrimSpace(string(out)), "\n")
	var totalRAM uint64
	for _, line := range lines {
		if strings.Contains(line, "=") {
			fields := strings.Split(line, "=")
			if len(fields) > 1 {
				capacity, _ := strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64)
				totalRAM += capacity
			}
		}
	}
	hardwareInfo.MemorieRAM = fmt.Sprintf("%d GB", totalRAM/(1024*1024*1024))

	// Obtine informatii despre stocare (inclusiv SSD-uri NVMe)
	cmd = exec.Command("powershell", "-Command", "Get-PhysicalDisk | Where-Object {$_.BusType -eq 'NVMe'} | ConvertTo-Json")
	out, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("eroare la executarea comenzii PowerShell: %w", err)
	}

	//  Verificam dacă rezultatul este un array JSON sau un singur obiect
	var disks []map[string]interface{}
	if strings.HasPrefix(strings.TrimSpace(string(out)), "[") {
		// Este un array JSON
		err = json.Unmarshal(out, &disks)
		if err != nil {
			return nil, fmt.Errorf("eroare la parsarea JSON (array): %w", err)
		}
	} else {
		// Este un singur obiect JSON
		var disk map[string]interface{}
		err = json.Unmarshal(out, &disk)
		if err != nil {
			return nil, fmt.Errorf("eroare la parsarea JSON (obiect): %w", err)
		}
		disks = append(disks, disk) // Adăugăm obiectul la un slice
	}
	if len(disks) > 0 {
		// Extragem informatiile pentru primul disc gasit
		disk := disks[0]
		hardwareInfo.TipStocare = disk["MediaType"].(string)

		// Convertim dimensiunea discului din bytes in GB
		sizeBytes := int64(disk["Size"].(float64))
		hardwareInfo.CapacitateHDD = fmt.Sprintf("%d GB", sizeBytes/(1024*1024*1024))
	} else {
		hardwareInfo.TipStocare = "N/A"
		hardwareInfo.CapacitateHDD = "N/A"
	}

	// Obtine informatii despre placa de baza
	cmd = exec.Command("cmd", "/c", "wmic baseboard get Manufacturer,Product /FORMAT:LIST")
	out, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("eroare la executarea comenzii 'wmic baseboard get ...': %w", err)
	}
	lines = strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "=") {
			fields := strings.Split(line, "=")
			switch strings.TrimSpace(fields[0]) {
			case "Manufacturer":
				hardwareInfo.PlacaDeBaza = strings.TrimSpace(fields[1])
			case "Product":
				hardwareInfo.PlacaDeBaza += " " + strings.TrimSpace(fields[1])
			}
		}
		break // Se obtin doar informatiile despre prima placa de baza
	}

	// Obtine informatii despre placa video
	cmd = exec.Command("cmd", "/c", "wmic path win32_videocontroller get Name /FORMAT:LIST")
	out, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("eroare la executarea comenzii 'wmic path win32_videocontroller get Name': %w", err)
	}
	lines = strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "=") {
			fields := strings.Split(line, "=")
			if len(fields) > 1 {
				hardwareInfo.PlacaVideo = strings.TrimSpace(fields[1])
			}
		}
		break // Se obtin doar informatiile despre prima placa video
	}

	return hardwareInfo, nil
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
	// Alte informatii despre program
}

// Functie pentru a obtine informatii despre programele instalate
func getInstalledPrograms() ([]ProgramInfo, error) {
	var programs []ProgramInfo

	// Implementati metoda pentru a obtine lista de programe instalate pe sistemul de operare
	// Iata un exemplu simplificat pentru Windows, care utilizeaza registrele de sistem
	// pentru a obtine informatiile necesare. Trebuie adaptata pentru diferite sisteme de operare.

	// Exemplu: Windows
	cmd := exec.Command("cmd", "/c", "wmic product get Name,Version,Vendor,InstallDate /FORMAT:LIST")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("eroare la executarea comenzii 'wmic product get ...': %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var program ProgramInfo
	for _, line := range lines {
		if strings.Contains(line, "=") {
			fields := strings.Split(line, "=")
			switch strings.TrimSpace(fields[0]) {
			case "Name":
				program.Nume = strings.TrimSpace(fields[1])
			case "Version":
				program.Versiune = strings.TrimSpace(fields[1])
			case "Vendor":
				program.Producator = strings.TrimSpace(fields[1])
			case "InstallDate":
				program.DataInstalare = strings.TrimSpace(fields[1])
			}
		} else {
			// Finalizam informatiile pentru un program si adaugam in lista
			if program.Nume != "" {
				programs = append(programs, program)
				program = ProgramInfo{}
			}
		}
	}
	// Adaugam ultimul program (daca exista)
	if program.Nume != "" {
		programs = append(programs, program)
	}

	return programs, nil
}

// Functie pentru a obtine informatii despre securitate
func getSecurityInfo() (string, error) {
	// Implementati metoda pentru a obtine informatii despre statusul de securitate al sistemului
	// Informatiile pot include statusul antivirusului, firewall-ului, etc.

	// Exemplu simplificat pentru Windows
	cmd := exec.Command("cmd", "/c", "wmic /namespace:\\\\root\\SecurityCenter2 path AntiVirusProduct get displayName /FORMAT:LIST")
	out, err := cmd.Output()
	if err != nil {
		return "N/A", fmt.Errorf("eroare la executarea comenzii 'wmic /namespace:\\\\root\\SecurityCenter2 path AntiVirusProduct get displayName': %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "=") {
			fields := strings.Split(line, "=")
			if len(fields) > 1 {
				return strings.TrimSpace(fields[1]), nil
			}
		}
	}
	return "N/A", nil
}

// Structura pentru informatii despre utilizator
type UserInfo struct {
	NumeUtilizator string `json:"nume_utilizator"`
	GrupUtilizator string `json:"grup_utilizator"`
	// Alte informatii despre utilizator
}

// Functie pentru a obtine informatii despre utilizatorul curent
func getCurrentUserInfo() (*UserInfo, error) {
	userInfo := &UserInfo{}

	// Implementati metoda pentru a obtine informatii despre utilizatorul curent
	// Aceasta poate include numele utilizatorului, grupul utilizatorului, etc.

	// Exemplu simplificat pentru Windows
	cmd := exec.Command("cmd", "/c", "whoami")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("eroare la executarea comenzii 'whoami': %w", err)
	}
	userInfo.NumeUtilizator = strings.TrimSpace(string(out))

	// Pentru grupul utilizatorului, se poate face o interogare suplimentara sau folosind pachete/librarii aditionale

	return userInfo, nil
}

// Structura pentru informatii live despre sistem
type LiveSystemInfo struct {
	UtilizareCPU      float64 `json:"utilizare_cpu"`
	UtilizareRAM      float64 `json:"utilizare_ram"`
	TraficTrimis      uint64  `json:"trafic_retea_bytes_trimisi"`
	TraficReceptionat uint64  `json:"trafic_retea_bytes_primiti"`
}

// Functie pentru a obtine informatii live despre sistem
func getLiveSystemInfo() (*LiveSystemInfo, error) {
	liveInfo := &LiveSystemInfo{}

	// CPU Usage
	cpuUsage, err := cpu.Percent(time.Second, true)
	if err != nil {
		return nil, fmt.Errorf("eroare la obținerea utilizării CPU: %w", err)
	}
	liveInfo.UtilizareCPU = cpuUsage[0]

	// RAM Usage
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("eroare la obținerea utilizării RAM: %w", err)
	}
	liveInfo.UtilizareRAM = memInfo.UsedPercent

	// Network Usage
	netIO, err := net.IOCounters(false)
	if err != nil {
		return nil, fmt.Errorf("eroare la obținerea utilizării rețelei: %w", err)
	}
	liveInfo.TraficReceptionat = netIO[0].BytesRecv
	liveInfo.TraficTrimis = netIO[0].BytesSent

	return liveInfo, nil
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

func updateDatabase(db *sql.DB, systemInfo map[string]interface{}, liveInfo *LiveSystemInfo, idStatie int) error {
	// Extrage informații din structura systemInfo
	osInfo := systemInfo["sistem_de_operare"].(*OSInfo)
	hardwareInfo := systemInfo["hardware"].(*HardwareInfo)
	softwareInfo := systemInfo["software"].(*SoftwareInfo)
	securityInfo := systemInfo["securitate"].(string)
	//userInfo := systemInfo["utilizator"].(*UserInfo)

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
	`, idStatie, strings.Split(hardwareInfo.PlacaDeBaza, " ")[0], hardwareInfo.Procesor, hardwareInfo.Nuclee,
		hardwareInfo.FireExecutie, hardwareInfo.Frecventa, hardwareInfo.MemorieRAM, hardwareInfo.TipStocare,
		hardwareInfo.CapacitateHDD, hardwareInfo.PlacaDeBaza, hardwareInfo.PlacaVideo,
		osInfo.Nume, osInfo.Versiune, osInfo.Arhitectura,
		osInfo.DataInstalarii, osInfo.Licenta, securityInfo)
	if err != nil {
		return fmt.Errorf("eroare la actualizarea metadatelor stației: %w", err)
	}

	// 4. Actualizare tabel 'software_instalat'
	for _, program := range softwareInfo.ProgrameInstalate {
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
	`, idStatie, liveInfo.UtilizareCPU, liveInfo.UtilizareRAM, liveInfo.TraficTrimis, liveInfo.TraficReceptionat)
	if err != nil {
		return fmt.Errorf("eroare la inserarea metricii stației: %w", err)
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

	// Creează fișierul JSON dacă nu există
	fileName := "live_data.json"
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		_, err = os.Create(fileName)
		if err != nil {
			fmt.Printf("Eroare la crearea fișierului JSON: %v\n", err)
			return
		}
	}
	for {
		// Creare structura pentru informațiile complete despre sistem
		systemInfo := make(map[string]interface{})

		// Obținere informații despre sistemul de operare, hardware, software etc.
		osInfo, err := getOSInfo()
		if err != nil {
			fmt.Printf("Eroare la obținerea informațiilor despre sistemul de operare: %v\n", err)
			// Gestionare eroare - puteți seta valori implicite sau să continuați cu precauție
			osInfo = &OSInfo{}
		}
		systemInfo["sistem_de_operare"] = osInfo

		hardwareInfo, err := getHardwareInfo()
		if err != nil {
			fmt.Printf("Eroare la obținerea informațiilor despre hardware: %v\n", err)
			hardwareInfo = &HardwareInfo{}
		}
		systemInfo["hardware"] = hardwareInfo

		installedPrograms, err := getInstalledPrograms()
		if err != nil {
			fmt.Printf("Eroare la obținerea informațiilor despre programele instalate: %v\n", err)
			installedPrograms = []ProgramInfo{}
		}
		softwareInfo := &SoftwareInfo{
			ProgrameInstalate: installedPrograms,
		}
		systemInfo["software"] = softwareInfo

		securityInfo, err := getSecurityInfo()
		if err != nil {
			fmt.Printf("Eroare la obținerea informațiilor despre securitate: %v\n", err)
			securityInfo = "N/A"
		}
		systemInfo["securitate"] = securityInfo

		userInfo, err := getCurrentUserInfo()
		if err != nil {
			fmt.Printf("Eroare la obținerea informațiilor despre utilizator: %v\n", err)
			userInfo = &UserInfo{}
		}
		systemInfo["utilizator"] = userInfo

		// Extrage informații live
		liveInfo, err := getLiveSystemInfo()
		if err != nil {
			fmt.Printf("Eroare la obținerea informațiilor live despre sistem: %v\n", err)
			// Gestionare eroare - puteți seta valori implicite sau să continuați cu precauție
			liveInfo = &LiveSystemInfo{}
		} else {
			fmt.Printf("--------------------\n")
			fmt.Printf("Informații Live:\n")
			fmt.Printf("Utilizare CPU: %.2f%%\n", liveInfo.UtilizareCPU)
			fmt.Printf("Utilizare RAM: %.2f%%\n", liveInfo.UtilizareRAM)
			fmt.Printf("Trafic Retea Trimis: %d\n", liveInfo.TraficTrimis)
			fmt.Printf("Trafic Retea Receptionat: %d\n", liveInfo.TraficReceptionat)
		}

		// Actualizează datele live în structura systemInfo
		systemInfo["utilizare_cpu"] = liveInfo.UtilizareCPU
		systemInfo["utilizare_ram"] = liveInfo.UtilizareRAM
		systemInfo["trafic_retea_bytes_trimisi"] = liveInfo.TraficTrimis
		systemInfo["trafic_retea_bytes_primiti"] = liveInfo.TraficReceptionat

		// Serializează toate datele în format JSON
		jsonData, err := json.MarshalIndent(systemInfo, "", "  ")
		if err != nil {
			fmt.Printf("Eroare la serializarea JSON: %v\n", err)
			continue // Trece la următoarea iterație a buclei
		}

		// Scrie datele JSON în fișier
		err = os.WriteFile(fileName, jsonData, 0644)
		if err != nil {
			fmt.Printf("Eroare la scrierea în fișierul JSON: %v\n", err)
			continue // Trece la următoarea iterație a buclei
		}
		// Actualizare baza de date
		err = updateDatabase(db, systemInfo, liveInfo, idStatie)
		if err != nil {
			fmt.Printf("Eroare la actualizarea bazei de date: %v\n", err)
		} else {
			fmt.Println("Baza de date actualizată cu succes!")
		}

		time.Sleep(10 * time.Second)
	}
}
