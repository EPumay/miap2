package DiskManagement

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"proyecto1/Structs"
	"proyecto1/Utilities"
	"strings"
	"time"
)

func Mkdisk(size int, fit string, unit string, path string) string {
	var respuesta string

	respuesta = "Comando: mkdisk\n"
	respuesta += "Tamaño: " + fmt.Sprint(size) + "\n"
	respuesta += "Ajuste: " + fit + "\n"
	respuesta += "Unidad: " + unit + "\n"
	respuesta += "Ruta: " + path + "\n"
	respuesta += "-------------------------------------\n"

	if fit != "bf" && fit != "wf" && fit != "ff" {
		fmt.Println("Error: Fit debe ser bf, wf or ff")
		respuesta = "Error: Fit debe ser bf, wf or ff"
		return respuesta
	}
	if size <= 0 {
		fmt.Println("Error: Size debe ser mayo a  0")
		respuesta = "Error: Size debe ser mayo a  0"
		return respuesta
	}
	if unit != "k" && unit != "m" {
		fmt.Println("Error: Las unidades validas son k o m")
		respuesta = "Error: Las unidades validas son k o m"
		return respuesta
	}

	/*
		Si el usuario especifica unit = "k" (Kilobytes), el tamaño se multiplica por 1024 para convertirlo a bytes.
		Si el usuario especifica unit = "m" (Megabytes), el tamaño se multiplica por 1024 * 1024 para convertirlo a MEGA bytes.
	*/
	// Asignar tamanio
	if unit == "k" {
		size = size * 1024
	} else {
		size = size * 1024 * 1024
	}

	// Crear el archivo
	err := Utilities.CreateFile(path)
	if err != nil {
		fmt.Println("Error: ", err)
		respuesta = "Error: " + err.Error()
		return respuesta
	}

	// Abrir el archivo
	file, err := Utilities.OpenFile(path)
	if err != nil {
		return err.Error()
	}

	//llenar el archivo con ceros
	datos := make([]byte, size)
	newErr := Utilities.WriteObject(file, datos, 0)
	if newErr != nil {
		fmt.Println("MKDISK Error: ", newErr)
		return "MKDISK Error: " + newErr.Error()
	}
	var newMBR Structs.MBR
	newMBR.MbrSize = int32(size)
	newMBR.Id = rand.Int31() // Numero random rand.Int31() genera solo números no negativos
	copy(newMBR.Fit[:], fit)
	ahora := time.Now()
	copy(newMBR.FechaC[:], ahora.Format("02/01/2006 15:04"))
	// Escribir el MBR en el archivo
	if err := Utilities.WriteObject(file, newMBR, 0); err != nil {
		return "ERROR"
	}
	// Cerrar el archivo
	defer file.Close()

	return respuesta
}

func Rmdisk(path string) (respuesta string) {
	fmt.Println("*************Inicio RMDISK*************")
	fmt.Println("Path: ", path)

	// Verificar si el archivo existe primero
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("Error: El archivo no existe")
		return "Error: El disco no existe"
	}

	// Eliminar el archivo
	err := os.Remove(path)
	if err != nil {
		fmt.Println("Error: ", err)
		return "Error: " + err.Error()
	}
	fmt.Println("*************Fin RMDISK*************")
	return "El disco ha sido eliminado"

}

func Fdisk(size int, path string, name string, unit string, type_ string, fit string) (respuesta string) {
	fmt.Println("*************Inicio FDISK*************")
	fmt.Println("Tamaño: ", size)
	fmt.Println("Unidad: ", unit)
	fmt.Println("Ruta: ", path)
	fmt.Println("Nombre: ", name)
	fmt.Println("Ajuste: ", fit)

	if unit == "k" {
		size = size * 1024
	} else if unit == "m" {
		size = size * 1024 * 1024
	}
	// Verificar si el archivo existe primero
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("Error: El Disco no existe")
		return "Error: El disco no existe"
	}

	// Abrir el archivo
	file, err := Utilities.OpenFile(path)
	if err != nil {
		return err.Error()
	}

	// Leer el MBR
	var TempMBR Structs.MBR
	err = Utilities.ReadObject(file, &TempMBR, 0)
	if err != nil {
		return "Error al leer el MBR"

	}
	var primaryCount, extendedCount, totalPartitions int
	var usedSpace int32 = 0

	Structs.PrintMBR(TempMBR)
	for i := 0; i < 4; i++ { //4 son las particiones primarias permitidas
		if TempMBR.Partitions[i].Size != 0 { //si la particion esta es uso, size es distinto de 0
			totalPartitions++                       //contador de particiones existentes
			usedSpace += TempMBR.Partitions[i].Size //suma el espacio usado

			if TempMBR.Partitions[i].Type[0] == 'p' {
				primaryCount++ //contador de particiones primarias
			} else if TempMBR.Partitions[i].Type[0] == 'e' {
				extendedCount++ //contador de particiones extendidas
			}
		}
		//no estan las logicas, porque solo pueden existir dentro de una extendida
	}

	// Validar que no se exceda el número máximo de particiones primarias y extendidas
	if totalPartitions >= 4 {
		fmt.Println("Error: No se pueden crear más de 4 particiones primarias o extendidas en total.")
		return "Error: No se pueden crear más de 4 particiones primarias o extendidas en total."
	}

	// Validar que no se pueda crear una partición lógica sin una extendida
	if type_ == "l" && extendedCount == 0 {
		fmt.Println("Error: No se puede crear una partición lógica sin una partición extendida.")
		return "Error: No se puede crear una partición lógica sin una partición extendida."
	}

	// Validar que el tamaño de la nueva partición no exceda el tamaño del disco
	if usedSpace+int32(size) > TempMBR.MbrSize {
		fmt.Println("Error: No hay suficiente espacio en el disco para crear esta partición.")
		return "Error: No hay suficiente espacio en el disco para crear esta partición."
	}
	// Determinar la posición de inicio de la nueva partición
	var gap int32 = int32(binary.Size(TempMBR))
	if totalPartitions > 0 {
		gap = TempMBR.Partitions[totalPartitions-1].Start + TempMBR.Partitions[totalPartitions-1].Size
	}

	for i := 0; i < 4; i++ {
		if TempMBR.Partitions[i].Size == 0 {
			if type_ == "p" || type_ == "e" {
				// Crear partición primaria o extendida
				TempMBR.Partitions[i].Size = int32(size)
				TempMBR.Partitions[i].Start = gap
				copy(TempMBR.Partitions[i].Name[:], name)
				copy(TempMBR.Partitions[i].Fit[:], fit)
				copy(TempMBR.Partitions[i].Status[:], "0")
				copy(TempMBR.Partitions[i].Type[:], type_)
				TempMBR.Partitions[i].Correlative = int32(totalPartitions + 1)

				if type_ == "e" {
					// Inicializar el primer EBR en la partición extendida
					ebrStart := gap // El primer EBR se coloca al inicio de la partición extendida
					ebr := Structs.EBR{
						Fit:   fit[0],
						Start: ebrStart,
						Size:  0,
						Next:  -1,
					}
					copy(ebr.Name[:], "")
					Utilities.WriteObject(file, ebr, int64(ebrStart))
				}

				break
			}
		}
	}
	// Manejar la creación de particiones lógicas dentro de una partición extendida
	if type_ == "l" {
		for i := 0; i < 4; i++ {
			if TempMBR.Partitions[i].Type[0] == 'e' {
				ebrPos := TempMBR.Partitions[i].Start
				var ebr Structs.EBR
				for {
					Utilities.ReadObject(file, &ebr, int64(ebrPos))
					if ebr.Next == -1 {
						break
					}
					ebrPos = ebr.Next
				}

				// Calcular la posición de inicio de la nueva partición lógica
				newEBRPos := ebr.Start + ebr.Size                            // El nuevo EBR se coloca después de la partición lógica anterior
				logicalPartitionStart := newEBRPos + int32(binary.Size(ebr)) // El inicio de la partición lógica es justo después del EBR

				// Ajustar el siguiente EBR
				ebr.Next = newEBRPos
				Utilities.WriteObject(file, ebr, int64(ebrPos))

				// Crear y escribir el nuevo EBR
				newEBR := Structs.EBR{
					Fit:   fit[0],
					Start: logicalPartitionStart,
					Size:  int32(size),
					Next:  -1,
				}
				copy(newEBR.Name[:], name)
				Utilities.WriteObject(file, newEBR, int64(newEBRPos))

				// Imprimir el nuevo EBR creado
				fmt.Println("Nuevo EBR creado:")
				Structs.PrintEBR(newEBR)
				fmt.Println("")

				// Imprimir todos los EBRs en la partición extendida
				fmt.Println("Imprimiendo todos los EBRs en la partición extendida:")
				ebrPos = TempMBR.Partitions[i].Start
				for {
					err := Utilities.ReadObject(file, &ebr, int64(ebrPos))
					if err != nil {
						fmt.Println("Error al leer EBR:", err)
						break
					}
					Structs.PrintEBR(ebr)
					if ebr.Next == -1 {
						break
					}
					ebrPos = ebr.Next
				}

				break
			}
		}
		fmt.Println("")
	}

	// Sobrescribir el MBR
	if err := Utilities.WriteObject(file, TempMBR, 0); err != nil {
		fmt.Println("Error: Could not write MBR to file")
		return "Error: Could not write MBR to file"
	}

	var TempMBR2 Structs.MBR
	// Leer el objeto nuevamente para verificar
	if err := Utilities.ReadObject(file, &TempMBR2, 0); err != nil {
		fmt.Println("Error: Could not read MBR from file after writing")
		return "Error: Could not read MBR from file after writing"
	}

	// Imprimir el objeto MBR actualizado
	Structs.PrintMBR(TempMBR2)

	// Cerrar el archivo binario
	defer file.Close()

	fmt.Println("======FIN FDISK======")
	fmt.Println("")

	respuesta = "Comando: FDISK\n"
	respuesta += "Tamaño: " + fmt.Sprint(size) + "\n"
	respuesta += "Unidad: " + unit + "\n"
	respuesta += "Ruta: " + path + "\n"
	respuesta += "Nombre: " + name + "\n"
	respuesta += "Ajuste: " + fit + "\n"
	respuesta += "-------------------------------------\n"
	respuesta += "Partición creada con éxito\n"
	respuesta += "-------------------------------------\n"
	respuesta += "Fin FDISK\n"
	return respuesta
}

type MountedPartition struct {
	Path     string
	Name     string
	Id       string
	Status   byte // 0 = No montada, 1 = Montada
	Loggedin bool
}

var mountedPartitions = make(map[string][]MountedPartition)

func GetMountedPartitions() map[string][]MountedPartition {
	return mountedPartitions
}

func Mount(path string, name string) (respuesta string) {
	file, err := Utilities.OpenFile(path)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el archivo")
		return "Error: No se pudo abrir el archivo"
	}

	defer file.Close()
	var TempMBR Structs.MBR
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		fmt.Println("Error: No se pudo leer el MBR")
		return "Error: No se pudo leer el MBR"
	}
	fmt.Println("Buscando particion con nombre: ", name)

	partitionFound := false
	var partition Structs.Partition
	var partitionIndex int

	nameBytes := [16]byte{}
	copy(nameBytes[:], name)

	for i := 0; i < 4; i++ {
		if TempMBR.Partitions[i].Type[0] == 'p' && bytes.Equal(TempMBR.Partitions[i].Name[:], nameBytes[:]) { //buscar particiciones primarias
			partition = TempMBR.Partitions[i] // asignar la particion encontrada la variable
			partitionIndex = i                // guardar el indice de la particion
			partitionFound = true             // cambiar el estado a true
			break
		}
	}
	if !partitionFound {
		fmt.Println("Error: Partición no encontrada o no es una partición primaria")
		return "Error: Partición no encontrada o no es una partición primaria"
	}

	// Verificar si la partición ya está montada
	if partition.Status[0] == '1' {
		fmt.Println("Error: La partición ya está montada")
		return "Error: La partición ya está montada"
	}

	// Generar el ID de la partición
	diskID := generateDiskID(path)

	// Verificar si ya se ha montado alguna partición de este disco
	mountedPartitionsInDisk := mountedPartitions[diskID]
	var letter byte

	if len(mountedPartitionsInDisk) == 0 {
		// Es un nuevo disco, asignar la siguiente letra disponible
		if len(mountedPartitions) == 0 {
			letter = 'a'
		} else {
			lastDiskID := getLastDiskID()
			lastLetter := mountedPartitions[lastDiskID][0].Id[len(mountedPartitions[lastDiskID][0].Id)-1]
			letter = lastLetter + 1
		}
	} else {
		// Utilizar la misma letra que las otras particiones montadas en el mismo disco
		letter = mountedPartitionsInDisk[0].Id[len(mountedPartitionsInDisk[0].Id)-1]
	}

	// Incrementar el número para esta partición
	carnet := "202112395"
	lastTwoDigits := carnet[len(carnet)-2:] // Obtener los últimos dos dígitos del carnet
	partitionID := fmt.Sprintf("%s%d%c", lastTwoDigits, partitionIndex+1, letter)

	// Actualizar el estado de la partición a montada y asignar el ID
	partition.Status[0] = '1'
	copy(partition.Id[:], partitionID)
	TempMBR.Partitions[partitionIndex] = partition
	mountedPartitions[diskID] = append(mountedPartitions[diskID], MountedPartition{
		Path:   path,
		Name:   name,
		Id:     partitionID,
		Status: '1',
	})
	Structs.AddMontadas(partitionID, path) // Agregar a la lista de montadas
	// Escribir el MBR actualizado al archivo
	if err := Utilities.WriteObject(file, TempMBR, 0); err != nil {
		fmt.Println("Error: No se pudo sobrescribir el MBR en el archivo")
		return "Error: No se pudo sobrescribir el MBR en el archivo"
	}

	fmt.Printf("Partición montada con ID: %s\n", partitionID)

	fmt.Println("")
	// Imprimir el MBR actualizado
	fmt.Println("MBR actualizado:")
	Structs.PrintMBR(TempMBR)
	fmt.Println("")
	respuesta += "-------------------------------------\n"
	respuesta += "Comando: mount\n"
	respuesta += "Ruta: " + path + "\n"
	respuesta += "Nombre: " + name + "\n"
	respuesta += "ID: " + partitionID + "\n"
	respuesta += "-------------------------------------\n"
	respuesta += "Partición montada con éxito\n"
	return respuesta

}

// Función para obtener el ID del último disco montado
func getLastDiskID() string {
	var lastDiskID string
	for diskID := range mountedPartitions {
		lastDiskID = diskID
	}
	return lastDiskID
}

func generateDiskID(path string) string {
	return strings.ToLower(path)
}

func Mounted() string {
	var result strings.Builder
	result.WriteString("\n=== Particiones Montadas ===\n")

	if len(mountedPartitions) == 0 {
		result.WriteString("No hay particiones montadas\n")
		return result.String() // Devuelve la cadena con el mensaje
	}

	// Encabezado de la tabla
	result.WriteString(strings.Repeat("-", 90) + "\n")
	result.WriteString(fmt.Sprintf("| %-10s | %-15s | %-15s | %-20s | %-6s | %-7s |\n",
		"Disco ID", "Nombre", "ID Partición", "Archivo", "Estado", "Sesión"))
	result.WriteString(strings.Repeat("-", 90) + "\n")

	// Recorre las particiones montadas
	for diskID, partitions := range mountedPartitions {
		shortDiskID := diskID
		if len(diskID) > 10 {
			shortDiskID = diskID[:7] + "..."
		}

		for _, partition := range partitions {
			filename := filepath.Base(partition.Path)
			shortName := partition.Name
			if len(shortName) > 15 {
				shortName = shortName[:12] + "..."
			}

			// Truncar filename
			if len(filename) > 20 {
				filename = filename[:17] + "..."
			}

			// Estado legible
			status := "Inactivo"
			if partition.Status == '1' {
				status = "Activo"
			}

			// Añadir la línea de la partición a la variable result
			result.WriteString(fmt.Sprintf("| %-10s | %-15s | %-15s | %-20s | %-6s |\n",
				shortDiskID,
				shortName,
				partition.Id,
				filename,
				status))
		}
	}
	result.WriteString(strings.Repeat("-", 90) + "\n")
	return result.String() // Devolver el contenido completo de la tabla como string
}

func Unmount(entrada []string) string {
	var respuesta string
	var id string

	for _, parametro := range entrada[1:] {
		tmp := strings.TrimRight(parametro, " ")
		valores := strings.Split(tmp, "=")

		if len(valores) != 2 {
			fmt.Println("ERROR UNMOUNT, valor desconocido de parametros ", valores[1])
			respuesta += "ERROR UNMOUNT, valor desconocido de parametros " + valores[1]
			//Si falta el valor del parametro actual lo reconoce como error e interrumpe el proceso
			return respuesta
		}

		if strings.ToLower(valores[0]) == "id" {
			id = strings.ToUpper(valores[1])
		} else {
			fmt.Println("UNMOUNT Error: Parametro desconocido: ", valores[0])
			return "UNMOUNT Error: Parametro desconocido: " + valores[0] //por si en el camino reconoce algo invalido de una vez se sale
		}
	}

	if id != "" {
		var pathDico string
		var registro int //registro a eliminar

		eliminar := false
		//BUsca en struck de particiones montadas el id ingresado
		for i, montado := range Structs.Montadas {
			if montado.Id == id {
				eliminar = true
				pathDico = montado.PathM
				registro = i
			}
		}

		if eliminar {
			Disco, err := Utilities.OpenFile(pathDico)
			if err != nil {
				return "ERROR UNMOUNT OPEN FILE " + err.Error()
			}

			var mbr Structs.MBR
			// Read object from bin file
			if err := Utilities.ReadObject(Disco, &mbr, 0); err != nil {
				return "ERROR UNMOUNT READ FILE " + err.Error()
			}

			//Encontrar la particion en el disco
			for i := 0; i < 4; i++ {
				identificador := Structs.GetId(string(mbr.Partitions[i].Id[:]))
				if identificador == id {
					name := Structs.GetName(string(mbr.Partitions[i].Name[:]))
					var unmount Structs.Partition

					//Eliminar el id usando el id de la variable unmount
					mbr.Partitions[i].Id = unmount.Id
					copy(mbr.Partitions[i].Status[:], "I")

					//sobreescribir el mbr para guardar los cambios
					if err := Utilities.WriteObject(Disco, mbr, 0); err != nil { //Sobre escribir el mbr
						return "ERROR UNMOUNT " + err.Error()
					}
					fmt.Println("Particion con nombre ", name, " desmontada correctamente")
					break //para que ya no siga recorriendo si ya encontro la particion independientemente si se pudo o no reducir
				}
			}

			//elimina el la particion montada del struck
			Structs.Montadas = append(Structs.Montadas[:registro], Structs.Montadas[registro+1:]...)

			for _, montada := range Structs.Montadas {
				fmt.Println("Id " + string(montada.Id) + ", Disco: " + montada.PathM + "\n")
				//partMontadas += "Id "+ string(montada.Id)+ ", Disco: "+ montada.PathM+"\n"
			}

			for i := 0; i < 4; i++ {
				estado := string(mbr.Partitions[i].Status[:])
				if estado == "A" {
					//tmpMontadas:= "Particion: " + strconv.Itoa(i) + ", name: " +string(mbr.Partitions[i].Name[:]) + ", status: "+string(mbr.Partitions[i].Status[:])+", id: "+string(mbr.Partitions[i].Id[:])+", tipo: "+string(mbr.Partitions[i].Type[:])+", correlativo: "+ strconv.Itoa(int(mbr.Partitions[i].Correlative)) + ", fit: "+string(mbr.Partitions[i].Fit[:])+ ", start: "+strconv.Itoa(int(mbr.Partitions[i].Start))+ ", size: "+strconv.Itoa(int(mbr.Partitions[i].Size))
					//partMontadas += Utilities.EliminartIlegibles(tmpMontadas)+"\n"
					fmt.Println("patcion: ", i, ", name: ", string(mbr.Partitions[i].Name[:]), ", status: "+string(mbr.Partitions[i].Status[:]))
				}
			}
		} else {
			fmt.Println("ERROR UNMOUNT: ID NO ENCONTRADO")
			return "ERROR UNMOUNT: ID NO ENCONTRADO"
		}

	} else {
		fmt.Println("ERROR UNMOUNT NO SE INGRESO PARAMETRO ID")
		return "ERROR UNMOUNT NO SE INGRESO PARAMETRO ID"
	}
	return respuesta
}
