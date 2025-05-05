package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"proyecto1/Analyzer"
	"strings"

	"github.com/rs/cors"
)

type Entrada struct {
	Text string `json:"text"`
}

type StatusResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func main() {
	//EndPoint
	http.HandleFunc("/analizar", getCadenaAnalizar)

	c := cors.Default()

	// Configurar el manejador HTTP con CORS
	handler := c.Handler(http.DefaultServeMux)

	fmt.Println("Servidor escuchando en http://localhost:8080")
	http.ListenAndServe(":8080", handler)

}

func getCadenaAnalizar(w http.ResponseWriter, r *http.Request) {
	var respuesta string
	w.Header().Set("Content-Type", "application/json")

	var status StatusResponse
	if r.Method == http.MethodPost {
		var entrada Entrada
		if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
			http.Error(w, "Error al decodificar JSON", http.StatusBadRequest)
			status = StatusResponse{Message: "Error al decodificar JSON", Type: "unsucces"}
			json.NewEncoder(w).Encode(status)
			return
		}
		lector := bufio.NewScanner(strings.NewReader(entrada.Text))
		for lector.Scan() {
			if lector.Text() != "" {
				linea := strings.Split(lector.Text(), "#") //comentarios
				if len(linea[0]) != 0 {
					respuesta += Analyzer.Analyze(linea[0]) + "\n"
				}
				//Comentarios
				if len(linea) > 1 && linea[1] != "" {
					fmt.Println("#" + linea[1] + "\n")
					respuesta += "#" + linea[1] + "\n"
				}
			}

		}

		w.WriteHeader(http.StatusOK)

		status = StatusResponse{Message: respuesta, Type: "succes"}
		json.NewEncoder(w).Encode(status)

	} else {
		//http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		status = StatusResponse{Message: "Metodo no permitido", Type: "unsucces"}
		json.NewEncoder(w).Encode(status)
	}
}
