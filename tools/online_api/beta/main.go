package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	connectedV2RayUsers map[string]time.Time // Armazenar usuários V2Ray conectados
	sshOnlineUsers      int                  // Contagem de usuários SSH
)

const (
	v2rayLogFile  = "C:\Users\Administrator\Documents\GitHub\SSH-T-PROJECT-TOOLS\tools\online_api\beta\log"
	v2rayAPIKey   = "b2c1f84a1d3e92f63e1d73c7e55b8a19a93d5b405c5d88f7f367e27c084df0a7"
	sshAPIKey     = "b2c1f84a1d3e92f63e1d73c7e55b8a19a93d5b405c5d88f7f367e27c084df0a7"
	sshOnlineFile = "online.txt"
)

func main() {
	connectedV2RayUsers = make(map[string]time.Time)

	// Inicializa as rotinas de monitoramento
	go monitorV2RayLogs()            // Atualizar V2Ray
	go cleanOldV2RayLogs()           // Limpar logs antigos V2Ray
	go updateSSHOnlinePeriodically() // Atualizar usuários SSH

	// Define as rotas do servidor
	http.HandleFunc("/online", authenticateToken(handleSSHUsers))         // Rota SSH
	http.HandleFunc("/online/v2ray", authenticateToken(handleV2RayUsers)) // Rota V2Ray

	log.Println("Servidor rodando na porta 2095...")
	log.Fatal(http.ListenAndServe(":2095", nil))
}

// Middleware para autenticar o token da API
func authenticateToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token != v2rayAPIKey && token != sshAPIKey {
			http.Error(w, "Acesso negado. Token inválido.", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// Função para exibir o número de usuários SSH
func handleSSHUsers(w http.ResponseWriter, r *http.Request) {
	response := map[string]int{"onlineUsers": sshOnlineUsers}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Função para exibir o número de usuários V2Ray
func handleV2RayUsers(w http.ResponseWriter, r *http.Request) {
	totalOnlineUsers := len(connectedV2RayUsers)
	response := map[string]int{"onlineUsers": totalOnlineUsers}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Função que monitora os logs do V2Ray e atualiza a lista de usuários conectados
func monitorV2RayLogs() {
	for {
		updateConnectedV2RayUsers()
		time.Sleep(10 * time.Second)
	}
}

// Função que atualiza a lista de usuários V2Ray conectados
func updateConnectedV2RayUsers() {
	content, err := ioutil.ReadFile(v2rayLogFile)
	if err != nil {
		log.Println("Erro ao ler o arquivo de log V2Ray:", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	currentTime := time.Now()

	uniqueUsers := make(map[string]struct{})
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		timestamp := extractTimestampFromLog(line)
		user := extractUserFromLog(line)

		if user != "" && currentTime.Sub(timestamp) <= 10*time.Minute {
			uniqueUsers[user] = struct{}{}
		}
	}

	connectedV2RayUsers = make(map[string]time.Time)
	for user := range uniqueUsers {
		connectedV2RayUsers[user] = currentTime
	}
}

// Função que limpa logs antigos do V2Ray (mais de 48 horas)
func cleanOldV2RayLogs() {
	for {
		time.Sleep(1 * time.Hour)
		threshold := time.Now().Add(-48 * time.Hour)

		for user, timestamp := range connectedV2RayUsers {
			if timestamp.Before(threshold) {
				delete(connectedV2RayUsers, user)
			}
		}
	}
}

// Função que extrai o email do log do V2Ray
func extractUserFromLog(line string) string {
	re := regexp.MustCompile(`email:\s*([\w._%+-]+@[\w.-]+\.[a-zA-Z]{2,})`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Função que extrai o timestamp do log do V2Ray
func extractTimestampFromLog(line string) time.Time {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return time.Time{}
	}
	timestamp, _ := time.Parse("2006/01/02 15:04:05", parts[0]+" "+parts[1])
	return timestamp
}

// Função que atualiza periodicamente os usuários SSH
func updateSSHOnlinePeriodically() {
	for {
		updateSSHOnlineUsers()
		time.Sleep(1 * time.Minute)
	}
}

// Função que atualiza o número de usuários SSH conectados
func updateSSHOnlineUsers() {
	cmd := exec.Command("sh", "-c", "ps -x | grep sshd | grep -v root | grep priv | wc -l")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Erro ao executar o comando SSH: %v", err)
		return
	}

	sshUsers := strings.TrimSpace(string(output))
	sshOnlineUsers, err = strconv.Atoi(sshUsers)
	if err != nil {
		log.Printf("Erro ao converter o número de usuários SSH: %v", err)
	}
}
