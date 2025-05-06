package main

import (
	"encoding/json"
	"net/http"
	"proyecto1/DiskManagement"
	"strings"
)

type ReadMBRParams struct {
	Path string `json:"path"`
}

func ReadMBRHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var params ReadMBRParams
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			http.Error(w, "Error al procesar la solictud", http.StatusBadRequest)
			return
		}

		if params.Path == "" {
			http.Error(w, "Error: Se requiere el path del disco", http.StatusBadRequest)
			return
		}

		partitions, err := DiskManagement.ListPartitions(params.Path)
		if err != nil {
			http.Error(w, "Error al leer las particiones", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(partitions)
	} else {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

type MkDiskParams struct {
	Size int    `json:"size"`
	Fit  string `json:"fit"`
	Unit string `json:"unit"`
	Path string `json:"path"`
}

// Handler para el comando mkdisk
func MkDiskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var params MkDiskParams

		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			http.Error(w, "Error al procesar la solicitud", http.StatusBadRequest)
			return
		}

		if params.Size <= 0 {
			http.Error(w, "El tamaño debe ser mayor a 0", http.StatusBadRequest)
			return
		}

		if params.Fit != "bf" && params.Fit != "ff" && params.Fit != "wf" {
			http.Error(w, "El ajuste debe ser 'bf', 'ff' o 'wf'", http.StatusBadRequest)
			return
		}

		if params.Unit != "k" && params.Unit != "m" {
			http.Error(w, "La unidad debe ser 'k' o 'm'", http.StatusBadRequest)
			return
		}

		if params.Path == "" {
			http.Error(w, "La ruta es requerida", http.StatusBadRequest)
			return
		}

		// Llamar a la función que ejecuta el mkdisk
		DiskManagement.Mkdisk(params.Size, params.Fit, params.Unit, params.Path)

		// Responder con éxito
		response := map[string]string{
			"message": "Disco creado exitosamente",
		}
		json.NewEncoder(w).Encode(response)
	} else {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

type FdiskParams struct {
	Size   int    `json:"size"`
	Path   string `json:"path"`
	Name   string `json:"name"`
	Unit   string `json:"unit"`
	Type   string `json:"type"`
	Fit    string `json:"fit"`
	Delete string `json:"delete"`
	Add    int    `json:"add"`
}

// Handler para el comando fdisk
func FdiskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var params FdiskParams

		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			http.Error(w, "Error al procesar la solicitud", http.StatusBadRequest)
			return
		}

		if params.Delete != "" {
			if params.Path == "" || params.Name == "" {
				http.Error(w, "Para eliminar una partición, se requiere 'path' y 'name'.", http.StatusBadRequest)
				return
			}
			lowercaseName := strings.ToLower(params.Name)
			DiskManagement.DeletePartition(params.Path, lowercaseName, params.Delete)

			response := map[string]string{
				"message": "Partición eliminada exitosamente",
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Si hay modificación de espacio
		if params.Add != 0 {
			if params.Path == "" || params.Name == "" {
				http.Error(w, "Para modificar una partición, se requiere 'path' y 'name'.", http.StatusBadRequest)
				return
			}

			lowercaseName := strings.ToLower(params.Name)
			err := DiskManagement.ModifyPartition(params.Path, lowercaseName, params.Add, params.Unit)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest) // Manejo de error
				return
			}

			response := map[string]string{
				"message": "Espacio de la partición modificado exitosamente",
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Creación de particiones
		if params.Size <= 0 {
			http.Error(w, "El tamaño debe ser mayor a 0", http.StatusBadRequest)
			return
		}

		if params.Path == "" {
			http.Error(w, "La ruta es requerida", http.StatusBadRequest)
			return
		}

		if params.Unit != "k" && params.Unit != "m" {
			http.Error(w, "La unidad debe ser 'k' o 'm'", http.StatusBadRequest)
			return
		}

		if params.Type != "p" && params.Type != "e" && params.Type != "l" {
			http.Error(w, "El tipo debe ser 'p', 'e', o 'l'", http.StatusBadRequest)
			return
		}

		if params.Fit != "b" && params.Fit != "f" && params.Fit != "w" {
			http.Error(w, "El ajuste debe ser 'b', 'f', o 'w'", http.StatusBadRequest)
			return
		}

		if params.Fit == "" {
			params.Fit = "w"
		}

		lowercaseName := strings.ToLower(params.Name)
		DiskManagement.Fdisk(params.Size, params.Path, lowercaseName, params.Unit, params.Type, params.Fit)

		response := map[string]string{
			"message": "Partición creada exitosamente",
		}
		json.NewEncoder(w).Encode(response)
	} else {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}
