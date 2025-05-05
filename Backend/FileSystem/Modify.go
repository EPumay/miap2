package FileSystem

import (
	"encoding/binary"
	"proyecto1/Structs"
	"proyecto1/Utilities"
)

func CommandCopy(path string, destino string) string {
	currentUser := Structs.UsuarioActual
	respuesta := ""
	//verify if user is logged in
	if !currentUser.Status {
		respuesta = "Error: Se requiere inicio de sesión \n"
		return respuesta
	}
	//open the binary file
	Disk, err := Utilities.OpenFile(currentUser.DiskPath)
	if err != nil {
		return "Error: No se pudo abrir el disco \n"
	}
	//read the MBR
	var mbr Structs.MBR
	if err := Utilities.ReadObject(Disk, &mbr, 0); err != nil {
		return "Error: No se pudo leer el disco \n"
	}

	copy := false
	part := -1

	//find the partition
	for i := 0; i < 4; i++ {
		partitionId := Structs.GetId(string(mbr.Partitions[i].Id[:]))
		if partitionId == currentUser.IdPart {
			part = i
			copy = true
			break
		}
	}

	if copy {
		var superBlock Structs.Superblock

		readError := Utilities.ReadObject(Disk, &superBlock, int64(mbr.Partitions[part].Start))
		if readError != nil {
			return "Error: Particion no formateada"
		}
		idFinalInode := BuscarInodo(0, destino, superBlock, Disk)
		var finalInode Structs.Inode
		Utilities.ReadObject(Disk, &finalInode, int64(superBlock.S_inode_start+(idFinalInode*int32(binary.Size(Structs.Inode{})))))
	}

	return respuesta

}
