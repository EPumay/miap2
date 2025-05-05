package DiskManagement

import (
	"encoding/binary"
	"fmt"
	"os"
	"proyecto1/Structs"
	"proyecto1/Utilities"
	"strings"
)

func DeletePartition(file *os.File, mbr *Structs.MBR, name string, deleteType string) string {
	found := false
	var partitionIndex int
	var partition Structs.Partition

	// Buscar la partición por nombre en las primarias/extendidas
	for i := 0; i < 4; i++ {
		if strings.Trim(string(mbr.Partitions[i].Name[:]), "\x00") == name {
			partition = mbr.Partitions[i]
			partitionIndex = i
			found = true
			break
		}
	}

	// Si no se encontró en las primarias, buscar en las lógicas (dentro de la extendida)
	if !found {
		for i := 0; i < 4; i++ {
			if mbr.Partitions[i].Type[0] == 'e' {
				ebrPos := mbr.Partitions[i].Start
				var ebr Structs.EBR
				var prevEBRPos int32 = -1
				var prevEBR Structs.EBR

				for ebrPos != -1 {
					Utilities.ReadObject(file, &ebr, int64(ebrPos))
					currentName := strings.Trim(string(ebr.Name[:]), "\x00")

					if currentName == name {
						// Eliminar la partición lógica
						if deleteType == "full" {
							// Sobrescribir el espacio con ceros
							Utilities.ClearSpace(file, int64(ebr.Start), int64(ebr.Size))
						}

						// Reenlazar los EBRs
						if prevEBRPos == -1 {
							// Es el primer EBR en la cadena
							if ebr.Next == -1 {
								// Era el único EBR
								mbr.Partitions[i].Size = 0 // Si era el único, podrías marcar la extendida como vacía
							} else {
								// El siguiente EBR se convierte en el primero
								var nextEBR Structs.EBR
								Utilities.ReadObject(file, &nextEBR, int64(ebr.Next))
								nextEBR.Start = mbr.Partitions[i].Start
								Utilities.WriteObject(file, nextEBR, int64(mbr.Partitions[i].Start))
							}
						} else {
							// Enlazar el EBR anterior con el siguiente
							prevEBR.Next = ebr.Next
							Utilities.WriteObject(file, prevEBR, int64(prevEBRPos))
						}

						// Actualizar el MBR
						Utilities.WriteObject(file, *mbr, 0)
						return "Partición lógica eliminada con éxito"
					}

					prevEBRPos = ebrPos
					prevEBR = ebr
					ebrPos = ebr.Next
				}
			}
		}
		return "Error: No se encontró la partición con el nombre especificado"
	}

	// Procesar eliminación de partición primaria/extendida
	if deleteType == "full" {
		// Sobrescribir el espacio con ceros
		Utilities.ClearSpace(file, int64(partition.Start), int64(partition.Size))
	}

	// Si es extendida, también eliminar todas las lógicas
	if partition.Type[0] == 'e' {
		ebrPos := partition.Start
		var ebr Structs.EBR
		for ebrPos != -1 {
			Utilities.ReadObject(file, &ebr, int64(ebrPos))
			if deleteType == "full" {
				Utilities.ClearSpace(file, int64(ebr.Start), int64(ebr.Size))
			}
			nextPos := ebr.Next
			ebr.Next = -1
			ebr.Size = 0
			Utilities.WriteObject(file, ebr, int64(ebrPos))
			ebrPos = nextPos
		}
	}

	// Eliminar la partición del MBR
	mbr.Partitions[partitionIndex] = Structs.Partition{}
	Utilities.WriteObject(file, *mbr, 0)

	return "Partición eliminada con éxito"
}

// Función para redimensionar una partición
func ResizePartition(file *os.File, mbr *Structs.MBR, name string, size int, operation string) string {
	// Buscar la partición por nombre
	var partition *Structs.Partition
	found := false

	for i := 0; i < 4; i++ {
		if strings.Trim(string(mbr.Partitions[i].Name[:]), "\x00") == name {
			partition = &mbr.Partitions[i]

			found = true
			break
		}
	}

	if !found {
		return "Error: No se encontró la partición con el nombre especificado"
	}

	// Calcular nuevo tamaño
	var newSize int32
	if operation == "+" {
		newSize = partition.Size + int32(size)
	} else if operation == "-" {
		newSize = partition.Size - int32(size)
		if newSize <= 0 {
			return "Error: El nuevo tamaño sería menor o igual a cero"
		}
	} else {
		return "Error: Operación no válida (debe ser '+' o '-')"
	}

	// Verificar espacio disponible
	if newSize > partition.Size {
		// Aumentar tamaño - verificar que haya espacio disponible después
		var nextPartitionStart int32 = mbr.MbrSize
		for i := 0; i < 4; i++ {
			if mbr.Partitions[i].Start > partition.Start && mbr.Partitions[i].Start < nextPartitionStart {
				nextPartitionStart = mbr.Partitions[i].Start
			}
		}

		availableSpace := nextPartitionStart - (partition.Start + partition.Size)
		if int32(size) > availableSpace {
			return "Error: No hay suficiente espacio contiguo para expandir la partición"
		}
	}

	// Actualizar tamaño
	partition.Size = newSize
	Utilities.WriteObject(file, *mbr, 0)

	// Si es extendida, ajustar las lógicas si es necesario
	if partition.Type[0] == 'e' {
		// Aquí podrías implementar lógica para ajustar las particiones lógicas
		// si estás reduciendo el tamaño de la extendida
	}

	return fmt.Sprintf("Tamaño de partición ajustado a %d bytes", newSize)
}

// Función original para crear particiones (separada para mejor organización)
func CreatePartition(file *os.File, TempMBR *Structs.MBR, path string, name string, type_ string, fit string, size int) string {
	var primaryCount, extendedCount, totalPartitions int
	var usedSpace int32 = 0

	Structs.PrintMBR(*TempMBR)
	for i := 0; i < 4; i++ {
		if TempMBR.Partitions[i].Size != 0 {
			totalPartitions++
			usedSpace += TempMBR.Partitions[i].Size

			if TempMBR.Partitions[i].Type[0] == 'p' {
				primaryCount++
			} else if TempMBR.Partitions[i].Type[0] == 'e' {
				extendedCount++
			}
		}
	}

	// Validaciones originales...
	if totalPartitions >= 4 {
		return "Error: No se pueden crear más de 4 particiones primarias o extendidas en total."
	}

	if type_ == "l" && extendedCount == 0 {
		return "Error: No se puede crear una partición lógica sin una partición extendida."
	}

	if usedSpace+int32(size) > TempMBR.MbrSize {
		return "Error: No hay suficiente espacio en el disco para crear esta partición."
	}

	// Determinar posición de inicio
	var gap int32 = int32(binary.Size(*TempMBR))
	if totalPartitions > 0 {
		gap = TempMBR.Partitions[totalPartitions-1].Start + TempMBR.Partitions[totalPartitions-1].Size
	}

	// Crear partición...
	for i := 0; i < 4; i++ {
		if TempMBR.Partitions[i].Size == 0 {
			if type_ == "p" || type_ == "e" {
				TempMBR.Partitions[i].Size = int32(size)
				TempMBR.Partitions[i].Start = gap
				copy(TempMBR.Partitions[i].Name[:], name)
				copy(TempMBR.Partitions[i].Fit[:], fit)
				copy(TempMBR.Partitions[i].Status[:], "0")
				copy(TempMBR.Partitions[i].Type[:], type_)
				TempMBR.Partitions[i].Correlative = int32(totalPartitions + 1)

				if type_ == "e" {
					ebrStart := gap
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

	// Manejar particiones lógicas...
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

				newEBRPos := ebr.Start + ebr.Size
				logicalPartitionStart := newEBRPos + int32(binary.Size(ebr))

				ebr.Next = newEBRPos
				Utilities.WriteObject(file, ebr, int64(ebrPos))

				newEBR := Structs.EBR{
					Fit:   fit[0],
					Start: logicalPartitionStart,
					Size:  int32(size),
					Next:  -1,
				}
				copy(newEBR.Name[:], name)
				Utilities.WriteObject(file, newEBR, int64(newEBRPos))
				break
			}
		}
	}

	// Escribir MBR actualizado
	if err := Utilities.WriteObject(file, *TempMBR, 0); err != nil {
		return "Error: Could not write MBR to file"
	}

	return "Partición creada con éxito"
}
