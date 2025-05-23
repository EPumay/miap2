package Utilities

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"proyecto1/Structs"
	"strings"
	"unicode"
)

// Funcion para crear un archivo binario
func CreateFile(name string) error {
	//Se asegura que el archivo existe
	dir := filepath.Dir(name)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Println("Err CreateFile dir==", err)
		return err
	}

	// Crear archivo
	if _, err := os.Stat(name); os.IsNotExist(err) {
		file, err := os.Create(name)
		if err != nil {
			fmt.Println("Err CreateFile create==", err)
			return err
		}
		defer file.Close()
	}
	return nil
}

// Funcion para abrir un archivo binario ead/write mode
func OpenFile(name string) (*os.File, error) {
	file, err := os.OpenFile(name, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Err OpenFile==", err)
		return nil, err
	}
	return file, nil
}

// ClearSpace sobrescribe una región del archivo con ceros
// file: Archivo a modificar
// start: Posición de inicio en bytes
// size: Tamaño del área a limpiar en bytes
func ClearSpace(file *os.File, start int64, size int64) error {
	// Crear un buffer de ceros (1MB para eficiencia)
	bufferSize := int64(1024 * 1024) // 1MB
	if size < bufferSize {
		bufferSize = size
	}
	zeros := make([]byte, bufferSize)

	// Posicionarse al inicio del área a limpiar
	_, err := file.Seek(start, 0)
	if err != nil {
		return err
	}

	// Escribir ceros en bloques
	remaining := size
	for remaining > 0 {
		writeSize := bufferSize
		if remaining < bufferSize {
			writeSize = remaining
		}

		_, err := file.Write(zeros[:writeSize])
		if err != nil {
			return err
		}

		remaining -= writeSize
	}

	// Sincronizar para asegurar que se escriba a disco
	err = file.Sync()
	if err != nil {
		return err
	}

	return nil
}

func WriteObject(file *os.File, data interface{}, position int64) error {
	file.Seek(position, 0)
	err := binary.Write(file, binary.LittleEndian, data)
	if err != nil {
		fmt.Println("Err WriteObject==", err)
		return err
	}
	return nil
}

func ReadObject(file *os.File, data interface{}, position int64) error {
	file.Seek(position, 0)
	err := binary.Read(file, binary.LittleEndian, data)
	if err != nil {
		fmt.Println("Err ReadObject==", err)
		return err
	}
	return nil
}

// Función para generar el reporte DISK en formato .dot
func GenerateDiskReport(mbr Structs.MBR, ebrs []Structs.EBR, outputPath string, file *os.File, totalDiskSize int32) error {
	// Crear la carpeta si no existe
	reportsDir := filepath.Dir(outputPath)
	err := os.MkdirAll(reportsDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Error al crear la carpeta de reportes: %v", err)
	}

	// Crear el archivo .dot donde se generara el reporte
	dotFilePath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".dot"
	fileDot, err := os.Create(dotFilePath)
	if err != nil {
		return fmt.Errorf("Error al crear el archivo .dot de reporte: %v", err)
	}
	defer fileDot.Close()

	// Iniciar el contenido del archivo en formato Graphviz (.dot)
	content := "digraph G {\n"
	content += "\tnode [shape=none];\n"
	content += "\tgraph [splines=false];\n"
	content += "\tsubgraph cluster_disk {\n"
	content += "\t\tlabel=\"Disco1.dsk\";\n"
	content += "\t\tstyle=rounded;\n"
	content += "\t\tcolor=black;\n"

	// Iniciar tabla para las particiones
	content += "\t\ttable [label=<\n\t\t\t<TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\" CELLPADDING=\"10\">\n"
	content += "\t\t\t<TR>\n"
	content += "\t\t\t<TD>MBR (159 bytes)</TD>\n"

	// Variables para el porcentaje y espacio libre
	var usedSpace int32 = 159 // Tamaño del MBR en bytes
	var freeSpace int32 = totalDiskSize - usedSpace

	for i := 0; i < 4; i++ {
		part := mbr.Partitions[i]
		if part.Size > 0 { // Si la partición tiene un tamaño valido
			percentage := float64(part.Size) / float64(totalDiskSize) * 100
			partName := strings.TrimRight(string(part.Name[:]), "\x00") // Limpiar el nombre de la partición

			if string(part.Type[:]) == "p" { // Partición primaria
				content += fmt.Sprintf("\t\t\t<TD>Primaria<br/>%s<br/>%.2f%% del disco</TD>\n", partName, percentage)
				usedSpace += part.Size
			} else if string(part.Type[:]) == "e" { // Partición extendida
				content += "\t\t\t<TD>\n"
				content += "\t\t\t\t<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n"
				content += fmt.Sprintf("\t\t\t\t<TR><TD COLSPAN=\"5\">Extendida</TD></TR>\n")

				// Leer los EBRs y agregar las particiones lógicas
				content += "\t\t\t\t<TR>\n"
				for _, ebr := range ebrs {
					logicalPercentage := float64(ebr.Size) / float64(totalDiskSize) * 100
					content += fmt.Sprintf("\t\t\t\t<TD>EBR (32 bytes)</TD>\n\t\t\t\t<TD>Lógica<br/>%.2f%% del disco</TD>\n", logicalPercentage)
					usedSpace += ebr.Size + 32 // Añadir el tamaño de la partición lógica y el EBR
				}
				content += "\t\t\t\t</TR>\n"
				content += "\t\t\t\t</TABLE>\n"
				content += "\t\t\t</TD>\n"
			}
		}
	}

	// Recalcular el espacio libre
	freeSpace = totalDiskSize - usedSpace
	freePercentage := float64(freeSpace) / float64(totalDiskSize) * 100

	// Agregar el espacio libre restante
	content += fmt.Sprintf("\t\t\t<TD>Libre<br/>%.2f%% del disco</TD>\n", freePercentage)
	content += "\t\t\t</TR>\n"
	content += "\t\t\t</TABLE>\n>];\n"
	content += "\t}\n"
	content += "}\n"

	// Escribir el contenido en el archivo .dot
	_, err = fileDot.WriteString(content)
	if err != nil {
		return fmt.Errorf("Error al escribir en el archivo .dot: %v", err)
	}

	fmt.Println("Reporte DISK generado exitosamente en:", dotFilePath)
	return nil
}

func GenerateMBRReport(mbr Structs.MBR, ebrs []Structs.EBR, outputPath string, file *os.File) error {
	// Crear la carpeta si no existe
	reportsDir := filepath.Dir(outputPath)
	err := os.MkdirAll(reportsDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Error al crear la carpeta de reportes: %v", err)
	}

	// Crear el archivo .dot donde se generará el reporte
	dotFilePath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".dot"
	fileDot, err := os.Create(dotFilePath)
	if err != nil {
		return fmt.Errorf("Error al crear el archivo .dot de reporte: %v", err)
	}
	defer fileDot.Close()

	// Iniciar el contenido del archivo en formato Graphviz (.dot)
	content := "digraph G {\n"
	content += "\tnode [fillcolor=lightyellow style=filled]\n"

	// Subgrafo del MBR
	content += fmt.Sprintf("\tsubgraph cluster_MBR {\n\t\tcolor=lightgrey fillcolor=lightblue label=\"MBR\nTamaño: %d\nFecha Creación: %s\nDisk Signature: %d\" style=filled\n",
		mbr.MbrSize, string(mbr.FechaC[:]), mbr.Id)

	// Recorrer las particiones del MBR en orden
	lastPartId := ""
	for i := 0; i < 4; i++ {
		part := mbr.Partitions[i]
		if part.Size > 0 { // Si la partición tiene un tamaño válido
			partName := strings.TrimRight(string(part.Name[:]), "\x00") // Limpiar el nombre de la partición
			partId := fmt.Sprintf("PART%d", i+1)
			content += fmt.Sprintf("\t\t%s [label=\"Partición %d\nStatus: %s\nType: %s\nFit: %s\nStart: %d\nSize: %d\nName: %s\" fillcolor=green shape=box style=filled]\n",
				partId, i+1, string(part.Status[:]), string(part.Type[:]), string(part.Fit[:]), part.Start, part.Size, partName)

			// Conectar la partición actual con la anterior de manera invisible para mantener el orden
			if lastPartId != "" {
				content += fmt.Sprintf("\t\t%s -> %s [style=invis]\n", lastPartId, partId)
			}
			lastPartId = partId

			// Si la partición es extendida, leer los EBRs
			if string(part.Type[:]) == "e" {
				content += fmt.Sprintf("\tsubgraph cluster_EBR%d {\n\t\tcolor=black fillcolor=lightpink label=\"Partición Extendida %d\" style=dashed\n", i+1, i+1)

				// Recolectamos todos los EBRs en orden
				ebrPos := part.Start
				var ebrList []Structs.EBR
				for {
					var ebr Structs.EBR
					err := ReadObject(file, &ebr, int64(ebrPos)) // Asegúrate de que la función ReadObject proviene de Utilities
					if err != nil {
						fmt.Println("Error al leer EBR:", err)
						break
					}
					ebrList = append(ebrList, ebr)

					// Si no hay más EBRs, salir del bucle
					if ebr.Next == -1 {
						break
					}

					// Mover a la siguiente posición de EBR
					ebrPos = ebr.Next
				}

				// Ahora agregamos los EBRs en orden correcto
				lastEbrId := ""
				for j, ebr := range ebrList {
					ebrName := strings.TrimRight(string(ebr.Name[:]), "\x00") // Limpiar el nombre del EBR
					ebrId := fmt.Sprintf("EBR%d", j+1)
					content += fmt.Sprintf("\t\t%s [label=\"EBR\nStart: %d\nSize: %d\nNext: %d\nName: %s\" fillcolor=lightpink shape=box style=filled]\n",
						ebrId, ebr.Start, ebr.Size, ebr.Next, ebrName)

					// Conectar el EBR actual con el anterior de manera invisible para mantener el orden
					if lastEbrId != "" {
						content += fmt.Sprintf("\t\t%s -> %s [style=invis]\n", lastEbrId, ebrId)
					}
					lastEbrId = ebrId
				}

				content += "\t}\n" // Cerrar el subgrafo de la partición extendida
			}
		}
	}

	content += "\t}\n" // Cerrar el subgrafo del MBR

	content += "}\n" // Cerrar el grafo principal

	// Escribir el contenido en el archivo .dot
	_, err = fileDot.WriteString(content)
	if err != nil {
		return fmt.Errorf("Error al escribir en el archivo .dot: %v", err)
	}

	fmt.Println("Reporte MBR generado exitosamente en:", dotFilePath)
	return nil
}

func Reporte(path string, contenido string) error {
	//asegurar la ruta
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Println("Error al crear el reporte, path: ", err)
		return err
	}
	// Abrir o crear un archivo para escritura
	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Error al crear el archivo:", err)
		return err
	}
	defer file.Close()

	// Escribir en el archivo
	_, err = file.WriteString(contenido)
	if err != nil {
		fmt.Println("Error al escribir en el archivo:", err)
		return err
	}

	return err
}

func RepGraphizMBR(path string, contenido string, nombre string) error {
	//asegurar la ruta
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Println("Error al crear el reporte, path: ", err)
		return err
	}
	// Abrir o crear un archivo para escritura
	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Error al crear el archivo:", err)
		return err
	}
	defer file.Close()

	// Escribir en el archivo
	_, err = file.WriteString(contenido)
	if err != nil {
		fmt.Println("Error al escribir en el archivo:", err)
		return err
	}

	rep2 := dir + "/" + nombre + ".png"
	cmd := exec.Command("dot", "-Tpng", path, "-o", rep2)
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error al generar el reporte PNG: %v", err)
	}

	return err
}

func EliminartIlegibles(entrada string) string {
	// Función de transformación que elimina caracteres no legibles
	transformFunc := func(r rune) rune {
		//unicode.IsPrint indica si es legible o no.
		//si el caracter se puede leer, lo regresa, de lo contrario devuekve -1
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}

	// Aplicar la función de transformación a la cadena de entrada
	salida := strings.Map(transformFunc, entrada)
	return salida
}

func FillWithZeros(file *os.File, start int32, size int32) error {
	// Posiciona el archivo al inicio del área que debe ser llenada
	file.Seek(int64(start), 0)

	// Crear un buffer lleno de ceros
	buffer := make([]byte, size)

	// Escribir los ceros en el archivo
	_, err := file.Write(buffer)
	if err != nil {
		fmt.Println("Error al llenar el espacio con ceros:", err)
		return err
	}

	fmt.Println("Espacio llenado con ceros desde el byte", start, "por", size, "bytes.")
	return nil
}

// Función para verificar que un bloque del archivo esté lleno de ceros
func VerifyZeros(file *os.File, start int32, size int32) {
	zeros := make([]byte, size)
	_, err := file.ReadAt(zeros, int64(start))
	if err != nil {
		fmt.Println("Error al leer la sección eliminada:", err)
		return
	}

	// Verificar si todos los bytes leídos son ceros
	isZeroFilled := true
	for _, b := range zeros {
		if b != 0 {
			isZeroFilled = false
			break
		}
	}

	if isZeroFilled {
		fmt.Println("La partición eliminada está completamente llena de ceros.")
	} else {
		fmt.Println("Advertencia: La partición eliminada no está completamente llena de ceros.")
	}
}
