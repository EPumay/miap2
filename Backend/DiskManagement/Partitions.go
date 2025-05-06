package DiskManagement

import (
	"fmt"
	"proyecto1/Structs"
	"proyecto1/Utilities"
	"strings"
)

func DeletePartition(path string, name string, delete_ string) (respuesta string) {
	fmt.Println("======Start DELETE PARTITION======")
	fmt.Println("Path:", path)
	fmt.Println("Name:", name)
	fmt.Println("Delete type:", delete_)

	// Abrir el archivo binario en la ruta proporcionada
	file, err := Utilities.OpenFile(path)
	if err != nil {
		fmt.Println("Error: Could not open file at path:", path)
		respuesta = "Error: Could not open file at path: " + path
		return respuesta
	}

	var TempMBR Structs.MBR
	// Leer el objeto desde el archivo binario
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		fmt.Println("Error: Could not read MBR from file")
		respuesta = "Error: Could not read MBR from file"
		return respuesta
	}

	// Buscar la partición por nombre
	found := false
	for i := 0; i < 4; i++ {
		// Limpiar los caracteres nulos al final del nombre de la partición
		partitionName := strings.TrimRight(string(TempMBR.Partitions[i].Name[:]), "\x00")
		if partitionName == name {
			found = true

			// Si es una partición extendida, eliminar las particiones lógicas dentro de ella
			if TempMBR.Partitions[i].Type[0] == 'e' {
				fmt.Println("Eliminando particiones lógicas dentro de la partición extendida...")
				respuesta = "Eliminando particiones lógicas dentro de la partición extendida..."
				ebrPos := TempMBR.Partitions[i].Start
				var ebr Structs.EBR
				for {
					err := Utilities.ReadObject(file, &ebr, int64(ebrPos))
					if err != nil {
						fmt.Println("Error al leer EBR:", err)
						respuesta += "Error al leer EBR: " + err.Error()
						break
					}
					// Detener el bucle si el EBR está vacío
					if ebr.Start == 0 && ebr.Size == 0 {
						fmt.Println("EBR vacío encontrado, deteniendo la búsqueda.")
						respuesta += "EBR vacío encontrado, deteniendo la búsqueda."
						break
					}
					// Depuración: Mostrar el EBR leído
					fmt.Println("EBR leído antes de eliminar:")
					Structs.PrintEBR(ebr)

					// Eliminar partición lógica
					if delete_ == "fast" {
						ebr = Structs.EBR{}                             // Resetear el EBR manualmente
						Utilities.WriteObject(file, ebr, int64(ebrPos)) // Sobrescribir el EBR reseteado
					} else if delete_ == "full" {
						Utilities.FillWithZeros(file, ebr.Start, ebr.Size)
						ebr = Structs.EBR{}                             // Resetear el EBR manualmente
						Utilities.WriteObject(file, ebr, int64(ebrPos)) // Sobrescribir el EBR reseteado
					}

					// Depuración: Mostrar el EBR después de eliminar
					fmt.Println("EBR después de eliminar:")

					Structs.PrintEBR(ebr)

					if ebr.Next == -1 {
						break
					}
					ebrPos = ebr.Next
				}
			}

			// Proceder a eliminar la partición (extendida, primaria o lógica)
			if delete_ == "fast" {
				// Eliminar rápido: Resetear manualmente los campos de la partición
				TempMBR.Partitions[i] = Structs.Partition{} // Resetear la partición manualmente
				fmt.Println("Partición eliminada en modo Fast.")
				respuesta += "Partición eliminada en modo Fast."
			} else if delete_ == "full" {
				// Eliminar completamente: Resetear manualmente y sobrescribir con '\0'
				start := TempMBR.Partitions[i].Start
				size := TempMBR.Partitions[i].Size
				TempMBR.Partitions[i] = Structs.Partition{} // Resetear la partición manualmente
				// Escribir '\0' en el espacio de la partición en el disco
				Utilities.FillWithZeros(file, start, size)
				fmt.Println("Partición eliminada en modo Full.")
				respuesta += "Partición eliminada en modo Full."
				// Leer y verificar si el área está llena de ceros
				Utilities.VerifyZeros(file, start, size)
			}
			break
		}
	}

	if !found {
		// Buscar particiones lógicas si no se encontró en el MBR
		fmt.Println("Buscando en particiones lógicas dentro de las extendidas...")
		for i := 0; i < 4; i++ {
			if TempMBR.Partitions[i].Type[0] == 'e' { // Solo buscar dentro de particiones extendidas
				ebrPos := TempMBR.Partitions[i].Start
				var ebr Structs.EBR
				for {
					err := Utilities.ReadObject(file, &ebr, int64(ebrPos))
					if err != nil {
						fmt.Println("Error al leer EBR:", err)
						break
					}

					// Depuración: Mostrar el EBR leído
					fmt.Println("EBR leído:")
					Structs.PrintEBR(ebr)

					logicalName := strings.TrimRight(string(ebr.Name[:]), "\x00")
					if logicalName == name {
						found = true
						// Eliminar la partición lógica
						if delete_ == "fast" {
							ebr = Structs.EBR{}                             // Resetear el EBR manualmente
							Utilities.WriteObject(file, ebr, int64(ebrPos)) // Sobrescribir el EBR reseteado
							fmt.Println("Partición lógica eliminada en modo Fast.")
							respuesta += "Partición lógica eliminada en modo Fast."
						} else if delete_ == "full" {
							Utilities.FillWithZeros(file, ebr.Start, ebr.Size)
							ebr = Structs.EBR{}                             // Resetear el EBR manualmente
							Utilities.WriteObject(file, ebr, int64(ebrPos)) // Sobrescribir el EBR reseteado
							Utilities.VerifyZeros(file, ebr.Start, ebr.Size)
							fmt.Println("Partición lógica eliminada en modo Full.")
							respuesta += "Partición lógica eliminada en modo Full."
						}
						break
					}

					if ebr.Next == -1 {
						break
					}
					ebrPos = ebr.Next
				}
			}
			if found {
				break
			}
		}
	}

	if !found {
		fmt.Println("Error: No se encontró la partición con el nombre:", name)
		respuesta = "Error: No se encontró la partición con el nombre: " + name
		return respuesta
	}

	// Sobrescribir el MBR
	if err := Utilities.WriteObject(file, TempMBR, 0); err != nil {
		fmt.Println("Error: Could not write MBR to file")
		respuesta = "Error: Could not write MBR to file"
		return respuesta
	}

	// Leer el MBR actualizado y mostrarlo
	fmt.Println("MBR actualizado después de la eliminación:")
	Structs.PrintMBR(TempMBR)

	// Si es una partición extendida, mostrar los EBRs actualizados
	for i := 0; i < 4; i++ {
		if TempMBR.Partitions[i].Type[0] == 'e' {
			fmt.Println("Imprimiendo EBRs actualizados en la partición extendida:")
			ebrPos := TempMBR.Partitions[i].Start
			var ebr Structs.EBR
			for {
				err := Utilities.ReadObject(file, &ebr, int64(ebrPos))
				if err != nil {
					fmt.Println("Error al leer EBR:", err)
					break
				}
				// Detener el bucle si el EBR está vacío
				if ebr.Start == 0 && ebr.Size == 0 {
					fmt.Println("EBR vacío encontrado, deteniendo la búsqueda.")
					break
				}
				// Depuración: Imprimir cada EBR leído
				fmt.Println("EBR leído después de actualización:")
				Structs.PrintEBR(ebr)
				if ebr.Next == -1 {
					break
				}
				ebrPos = ebr.Next
			}
		}
	}

	// Cerrar el archivo binario
	defer file.Close()

	fmt.Println("======FIN DELETE PARTITION======")
	return respuesta
}

func ModifyPartition(path string, name string, add int, unit string) error {
	fmt.Println("======Start MODIFY PARTITION======")
	// Abrir el archivo binario en la ruta proporcionada
	file, err := Utilities.OpenFile(path)
	if err != nil {
		fmt.Println("Error: Could not open file at path:", path)
		return err
	}
	defer file.Close()

	// Leer el MBR
	var TempMBR Structs.MBR
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		fmt.Println("Error: Could not read MBR from file")
		return err
	}

	// Imprimir MBR antes de modificar
	fmt.Println("MBR antes de la modificación:")
	Structs.PrintMBR(TempMBR)

	// Buscar la partición por nombre
	var foundPartition *Structs.Partition
	var partitionType byte

	// Revisar si la partición es primaria o extendida
	for i := 0; i < 4; i++ {
		partitionName := strings.TrimRight(string(TempMBR.Partitions[i].Name[:]), "\x00")
		if partitionName == name {
			foundPartition = &TempMBR.Partitions[i]
			partitionType = TempMBR.Partitions[i].Type[0]
			break
		}
	}

	// Si no se encuentra en las primarias/extendidas, buscar en las particiones lógicas
	if foundPartition == nil {
		for i := 0; i < 4; i++ {
			if TempMBR.Partitions[i].Type[0] == 'e' {
				ebrPos := TempMBR.Partitions[i].Start
				var ebr Structs.EBR
				for {
					if err := Utilities.ReadObject(file, &ebr, int64(ebrPos)); err != nil {
						fmt.Println("Error al leer EBR:", err)
						return err
					}

					ebrName := strings.TrimRight(string(ebr.Name[:]), "\x00")
					if ebrName == name {
						partitionType = 'l' // Partición lógica
						foundPartition = &Structs.Partition{
							Start: ebr.Start,
							Size:  ebr.Size,
						}
						break
					}

					// Continuar buscando el siguiente EBR
					if ebr.Next == -1 {
						break
					}
					ebrPos = ebr.Next
				}
				if foundPartition != nil {
					break
				}
			}
		}
	}

	// Verificar si la partición fue encontrada
	if foundPartition == nil {
		fmt.Println("Error: No se encontró la partición con el nombre:", name)
		return nil // Salir si no se encuentra la partición
	}

	// Convertir unidades a bytes
	var addBytes int
	if unit == "k" {
		addBytes = add * 1024
	} else if unit == "m" {
		addBytes = add * 1024 * 1024
	} else {
		fmt.Println("Error: Unidad desconocida, debe ser 'k' o 'm'")
		return nil // Salir si la unidad no es válida
	}

	// Flag para saber si continuar o no
	var shouldModify = true

	// Comprobar si es posible agregar o quitar espacio
	if add > 0 {
		// Agregar espacio: verificar si hay suficiente espacio libre después de la partición
		nextPartitionStart := foundPartition.Start + foundPartition.Size
		if partitionType == 'l' {
			// Para particiones lógicas, verificar con el siguiente EBR o el final de la partición extendida
			for i := 0; i < 4; i++ {
				if TempMBR.Partitions[i].Type[0] == 'e' {
					extendedPartitionEnd := TempMBR.Partitions[i].Start + TempMBR.Partitions[i].Size
					if nextPartitionStart+int32(addBytes) > extendedPartitionEnd {
						fmt.Println("Error: No hay suficiente espacio libre dentro de la partición extendida")
						shouldModify = false
					}
					break
				}
			}
		} else {
			// Para primarias o extendidas
			if nextPartitionStart+int32(addBytes) > TempMBR.MbrSize {
				fmt.Println("Error: No hay suficiente espacio libre después de la partición")
				shouldModify = false
			}
		}
	} else {
		// Quitar espacio: verificar que no se reduzca el tamaño por debajo de 0
		if foundPartition.Size+int32(addBytes) < 0 {
			fmt.Println("Error: No es posible reducir la partición por debajo de 0")
			shouldModify = false
		}
	}

	// Solo modificar si no hay errores
	if shouldModify {
		foundPartition.Size += int32(addBytes)
	} else {
		fmt.Println("No se realizaron modificaciones debido a un error.")
		return nil // Salir si hubo un error
	}

	// Si es una partición lógica, sobrescribir el EBR
	if partitionType == 'l' {
		ebrPos := foundPartition.Start
		var ebr Structs.EBR
		if err := Utilities.ReadObject(file, &ebr, int64(ebrPos)); err != nil {
			fmt.Println("Error al leer EBR:", err)
			return err
		}

		// Actualizar el tamaño en el EBR y escribirlo de nuevo
		ebr.Size = foundPartition.Size
		if err := Utilities.WriteObject(file, ebr, int64(ebrPos)); err != nil {
			fmt.Println("Error al escribir el EBR actualizado:", err)
			return err
		}

		// Imprimir el EBR modificado
		fmt.Println("EBR modificado:")
		Structs.PrintEBR(ebr)
	}

	// Sobrescribir el MBR actualizado
	if err := Utilities.WriteObject(file, TempMBR, 0); err != nil {
		fmt.Println("Error al escribir el MBR actualizado:", err)
		return err
	}

	// Imprimir el MBR modificado
	fmt.Println("MBR después de la modificación:")
	Structs.PrintMBR(TempMBR)

	fmt.Println("======END MODIFY PARTITION======")
	return nil
}

func ListPartitions(path string) ([]PartitionInfo, error) {
	// Abrir el archivo binario
	file, err := Utilities.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error al abrir el archivo: %v", err)
	}
	defer file.Close()
	var mbr Structs.MBR
	err = Utilities.ReadObject(file, &mbr, 0) // Leer desde la posición 0
	if err != nil {
		return nil, fmt.Errorf("Error al leer el MBR: %v", err)
	}
	var partitions []PartitionInfo
	for _, partition := range mbr.Partitions {
		if partition.Size > 0 {
			partitionName := strings.TrimRight(string(partition.Name[:]), "\x00")

			partitions = append(partitions, PartitionInfo{
				Name:   partitionName,
				Type:   strings.TrimRight(string(partition.Type[:]), "\x00"),
				Start:  partition.Start,
				Size:   partition.Size,
				Status: strings.TrimRight(string(partition.Status[:]), "\x00"),
			})
		}
	}

	return partitions, nil
}

type PartitionInfo struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Start  int32  `json:"start"`
	Size   int32  `json:"size"`
	Status string `json:"status"`
}
