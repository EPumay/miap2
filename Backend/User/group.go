package User

import (
	"encoding/binary"
	"fmt"
	"proyecto1/Structs"
	"proyecto1/Utilities"
	"strconv"
	"strings"
)

func Mkgrp(entrada []string) string {
	var respuesta string
	var name string
	UsuarioA := Structs.UsuarioActual

	if !UsuarioA.Status {
		respuesta += "ERROR MKGRP: NO HAY SESION INICIADA" + "\n"
		respuesta += "POR FAVOR INICIAR SESION PARA CONTINUAR" + "\n"
		return respuesta
	}

	for _, parametro := range entrada[1:] {
		tmp := strings.TrimRight(parametro, " ")
		valores := strings.Split(tmp, "=")

		if len(valores) != 2 {
			fmt.Println("ERROR MKGRP, valor desconocido de parametros ", valores[1])
			respuesta += "ERROR MKGRP, valor desconocido de parametros " + valores[1] + "\n"
			//Si falta el valor del parametro actual lo reconoce como error e interrumpe el proceso
			return respuesta
		}
		tmp = strings.TrimRight(valores[1], "")
		valores[1] = tmp

		//********************  NAME *****************
		if strings.ToLower(valores[0]) == "name" {
			tmp = strings.ReplaceAll(valores[1], "\"", "")
			name = (tmp)
			//validar maximo 10 caracteres
			if len(name) > 10 {
				fmt.Println("MKGRP ERROR: name debe tener maximo 10 caracteres")
				return "ERROR MKGRP: name debe tener maximo 10 caracteres"
			}
			//******************* ERROR EN LOS PARAMETROS *************
		} else {
			fmt.Println("LOGIN ERROR: Parametro desconocido: ", valores[0])
			//por si en el camino reconoce algo invalido de una vez se sale
			return "LOGIN ERROR: Parametro desconocido: " + valores[0] + "\n"
		}
	}

	if UsuarioA.Nombre == "root" {
		file, err := Utilities.OpenFile(UsuarioA.DiskPath)
		if err != nil {
			return "ERROR REP SB OPEN FILE " + err.Error() + "\n"
		}

		var mbr Structs.MBR
		// Read object from bin file
		if err := Utilities.ReadObject(file, &mbr, 0); err != nil {
			return "ERROR REP SB READ FILE " + err.Error() + "\n"
		}

		// Close bin file
		defer file.Close()

		//Encontrar la particion correcta
		AddNewUser := false
		part := -1
		for i := 0; i < 4; i++ {
			identificador := Structs.GetId(string(mbr.Partitions[i].Id[:]))
			if identificador == UsuarioA.IdPart {
				part = i
				AddNewUser = true
				break //para que ya no siga recorriendo si ya encontro la particion independientemente si se pudo o no reducir
			}
		}

		if AddNewUser {
			var superBloque Structs.Superblock
			errREAD := Utilities.ReadObject(file, &superBloque, int64(mbr.Partitions[part].Start))
			if errREAD != nil {
				fmt.Println("REP Error. Particion sin formato")
				return "REP Error. Particion sin formato" + "\n"
			}

			var inodo Structs.Inode
			//Le agrego una structura de inodo para ver el user.txt que esta en el primer inodo del sb
			Utilities.ReadObject(file, &inodo, int64(superBloque.S_inode_start+int32(binary.Size(Structs.Inode{}))))

			//leer los datos del user.txt
			var contenido string
			var fileBlock Structs.Fileblock
			var idFb int32 //id/numero de ultimo fileblock para trabajar sobre ese
			for _, item := range inodo.I_block {
				if item != -1 {
					Utilities.ReadObject(file, &fileBlock, int64(superBloque.S_block_start+(item*int32(binary.Size(Structs.Fileblock{})))))
					contenido += string(fileBlock.B_content[:])
					idFb = item
				}
			}

			lineaID := strings.Split(contenido, "\n")

			//Verificar si el grupo ya existe
			for _, registro := range lineaID[:len(lineaID)-1] {
				datos := strings.Split(registro, ",")
				if len(datos) == 3 {
					if datos[2] == name {
						fmt.Println("MKGRP ERROR: El grupo ya existe")
						return "MKGRP ERROR: El grupo ya existe"
					}
				}
			}

			//Buscar el ultimo ID activo desde el ultimo hasta el primero (ignorando los eliminado (0))
			//desde -2 porque siempre se crea un salto de linea al final generando una linea vacia al final del arreglo
			id := -1        //para guardar el nuevo ID
			var errId error //para la conversion a numero del ID
			for i := len(lineaID) - 2; i >= 0; i-- {
				registro := strings.Split(lineaID[i], ",")
				//valido que sea un grupo
				if registro[1] == "G" {
					//valido que el id sea distinto a 0 (eliminado)
					if registro[0] != "0" {
						//convierto el id en numero para sumarle 1 y crear el nuevo id
						id, errId = strconv.Atoi(registro[0])
						if errId != nil {
							fmt.Println("MKGRP ERROR: No se pudo obtener un nuevo id para el nuevo grupo")
							return "MKGRP ERROR: No se pudo obtener un nuevo id para el nuevo grupo"
						}
						id++
						break
					}
				}
			}

			//valido que se haya encontrado un nuevo id
			if id != -1 {
				contenidoActual := string(fileBlock.B_content[:])
				posicionNulo := strings.IndexByte(contenidoActual, 0)
				data := fmt.Sprintf("%d,G,%s\n", id, name)
				//Aseguro que haya al menos un byte libre
				if posicionNulo != -1 {
					libre := 64 - (posicionNulo + len(data))
					if libre > 0 {
						copy(fileBlock.B_content[posicionNulo:], []byte(data))
						//Escribir el fileblock con espacio libre
						Utilities.WriteObject(file, fileBlock, int64(superBloque.S_block_start+(idFb*int32(binary.Size(Structs.Fileblock{})))))
					} else {
						//Si es 0 (quedó exacta), entra aqui y crea un bloque vacío que podrá usarse para el proximo registro
						data1 := data[:len(data)+libre]
						//Ingreso lo que cabe en el bloque actual
						copy(fileBlock.B_content[posicionNulo:], []byte(data1))
						Utilities.WriteObject(file, fileBlock, int64(superBloque.S_block_start+(idFb*int32(binary.Size(Structs.Fileblock{})))))

						//Creo otro fileblock para el resto de la informacion
						guardoInfo := true

						for i, item := range inodo.I_block {
							//i es el indice en el arreglo inodo.Iblock
							// DIferencia i/item:  inodo.I_block[i] = item
							if item == -1 {
								guardoInfo = false
								//agrego el apuntador del bloque al inodo
								inodo.I_block[i] = superBloque.S_first_blo
								//actualizo el superbloque
								superBloque.S_free_blocks_count -= 1
								superBloque.S_first_blo += 1
								data2 := data[len(data)+libre:]
								//crear nuevo fileblock
								var newFileBlock Structs.Fileblock
								copy(newFileBlock.B_content[:], []byte(data2))

								//escribir las estructuras para guardar los cambios
								// Escribir el superbloque
								Utilities.WriteObject(file, superBloque, int64(mbr.Partitions[part].Start))

								//escribir el bitmap de bloques (se uso un bloque). inodo.I_block[i] contiene el numero de bloque que se uso
								Utilities.WriteObject(file, byte(1), int64(superBloque.S_bm_block_start+inodo.I_block[i]))

								//escribir inodes (es el inodo 1, porque es donde esta users.txt)
								Utilities.WriteObject(file, inodo, int64(superBloque.S_inode_start+int32(binary.Size(Structs.Inode{}))))

								//Escribir bloques
								Utilities.WriteObject(file, newFileBlock, int64(superBloque.S_block_start+(inodo.I_block[i]*int32(binary.Size(Structs.Fileblock{})))))
								break
							}
						}

						if guardoInfo {
							fmt.Println("MKGRP ERROR: Espacio insuficiente para nuevo registro")
							return "MKGRP ERROR: Espacio insuficiente para nuevo registro. "
						}
					}

					fmt.Println("Se ha agregado el grupo '" + name + "' exitosamente. ")
					respuesta = "Se ha agregado el grupo '" + name + "' exitosamente."
					for k := 0; k < len(lineaID)-1; k++ {
						fmt.Println(lineaID[k])
					}
					return respuesta
				}
			}
			//FIn Add new Usuario
		} else {
			fmt.Println("ERROR INESPERADO CON LA PARCION EN MKGRP")
			respuesta += "ERROR INESPERADO CON LA PARCION EN MKGRP"
		}

	} else {
		fmt.Println("ERROR FALTA DE PERMISOS, NO ES EL USUARIO ROOT")
		respuesta += "ERROR MKGRO: ESTE USUARIO NO CUENTA CON LOS PERMISOS PARA REALIZAR ESTA ACCION"
	}

	return respuesta
}

func Rmgrp(entrada []string) string {
	var respuesta string
	var name string
	UsuarioA := Structs.UsuarioActual

	if !UsuarioA.Status {
		respuesta += "ERROR RMGRP: NO HAY SECION INICIADA" + "\n"
		respuesta += "POR FAVOR INICIAR SESION PARA CONTINUAR" + "\n"
		return respuesta
	}

	for _, parametro := range entrada[1:] {
		tmp := strings.TrimRight(parametro, " ")
		valores := strings.Split(tmp, "=")

		if len(valores) != 2 {
			fmt.Println("ERROR RMGRP, valor desconocido de parametros ", valores[1])
			respuesta += "ERROR RMGRP, valor desconocido de parametros " + valores[1] + "\n"
			//Si falta el valor del parametro actual lo reconoce como error e interrumpe el proceso
			return respuesta
		}

		//********************  NAME *****************
		if strings.ToLower(valores[0]) == "name" {
			name = (valores[1])
			//validar maximo 10 caracteres
			if len(name) > 10 {
				fmt.Println("RMGRP ERROR: name debe tener maximo 10 caracteres")
				return "ERROR RMGRP: name debe tener maximo 10 caracteres"
			}
			//******************* ERROR EN LOS PARAMETROS *************
		} else {
			fmt.Println("RMGRP ERROR: Parametro desconocido: ", valores[0])
			//por si en el camino reconoce algo invalido de una vez se sale
			return "RMGRP ERROR: Parametro desconocido: " + valores[0] + "\n"
		}
	}

	if UsuarioA.Nombre == "root" {
		file, err := Utilities.OpenFile(UsuarioA.DiskPath)
		if err != nil {
			return "RMGRP ERRORSB OPEN FILE " + err.Error() + "\n"
		}

		var mbr Structs.MBR
		// Read object from bin file
		if err := Utilities.ReadObject(file, &mbr, 0); err != nil {
			return "RMGRP ERRORSB READ FILE " + err.Error() + "\n"
		}

		// Close bin file
		defer file.Close()

		//Encontrar la particion correcta
		delete := false
		part := -1 //particion a utilizar y modificar
		for i := 0; i < 4; i++ {
			identificador := Structs.GetId(string(mbr.Partitions[i].Id[:]))
			if identificador == UsuarioA.IdPart {
				part = i
				delete = true
				break //para que ya no siga recorriendo si ya encontro la particion independientemente si se pudo o no reducir
			}
		}

		if delete {
			var superBloque Structs.Superblock
			errREAD := Utilities.ReadObject(file, &superBloque, int64(mbr.Partitions[part].Start))
			if errREAD != nil {
				fmt.Println("RMGRP ERROR. Particion sin formato")
				return "RMGRP ERROR. Particion sin formato" + "\n"
			}

			var inodo Structs.Inode
			//Le agrego una structura de inodo para ver el user.txt que esta en el primer inodo del sb
			Utilities.ReadObject(file, &inodo, int64(superBloque.S_inode_start+int32(binary.Size(Structs.Inode{}))))

			//leer los datos del user.txt
			var contenido string
			var fileBlock Structs.Fileblock
			for _, item := range inodo.I_block {
				if item != -1 {
					Utilities.ReadObject(file, &fileBlock, int64(superBloque.S_block_start+(item*int32(binary.Size(Structs.Fileblock{})))))
					contenido += string(fileBlock.B_content[:])
				}
			}

			lineaID := strings.Split(contenido, "\n")
			modificarUs := false
			for k := 0; k < len(lineaID); k++ {
				datos := strings.Split(lineaID[k], ",")
				if len(datos) == 3 {
					if datos[2] == name {
						//por si ya estaba eliminado
						if datos[0] != "0" {
							modificarUs = true
							datos[0] = "0"
							lineaID[k] = datos[0] + "," + datos[1] + "," + datos[2]
						} else {
							fmt.Println("ERROR RMGRP ESTE GRUPO YA FUE ELIMINADO PREVIAMENTE")
							return "ERROR RMGRP ESTE GRUPO YA FUE ELIMINADO PREVIAMENTE"
						}
					}
				}
			}

			if modificarUs {
				//MODIFICA LOS USUARIOS DE ESE GRUPO
				for k := 0; k < len(lineaID); k++ {
					datos := strings.Split(lineaID[k], ",")
					if len(datos) == 5 {
						if datos[2] == name {
							if datos[0] != "0" {
								datos[0] = "0"
								lineaID[k] = datos[0] + "," + datos[1] + "," + datos[2] + "," + datos[3] + "," + datos[4]
							}
						}
					}
				}

				mod := ""
				for _, reg := range lineaID {
					mod += reg + "\n"
				}

				inicio := 0
				var fin int
				if len(mod) > 64 {
					//si el contenido es mayor a 64 bytes. la primera vez termina en 64
					fin = 64
				} else {
					//termina en el tamaño del contenido. Solo habra un fileblock porque ocupa menos de la capacidad de uno
					fin = len(mod)
				}

				for _, newItem := range inodo.I_block {
					if newItem != -1 {
						//tomo 64 bytes de la cadena o los bytes que queden
						data := mod[inicio:fin]
						//Modifico y guardo el bloque actual
						var newFileBlock Structs.Fileblock
						copy(newFileBlock.B_content[:], []byte(data))
						Utilities.WriteObject(file, newFileBlock, int64(superBloque.S_block_start+(newItem*int32(binary.Size(Structs.Fileblock{})))))
						//muevo a los siguientes 64 bytes de la cadena (o los que falten)
						inicio = fin
						calculo := len(mod[fin:]) //tamaño restante de la cadena
						//else if
						if calculo > 64 {
							fin += 64
						} else {
							fin += calculo
						}
					}
				}

				fmt.Println("El grupo '" + name + "' fue eliminado con extiso")
				respuesta += "El grupo '" + name + "' fue eliminado con extiso"
				for k := 0; k < len(lineaID)-1; k++ {
					fmt.Println(lineaID[k])
				}
				return respuesta
			}
		}

	} else {
		fmt.Println("ERROR FALTA DE PERMISOS, NO ES EL USUARIO ROOT")
		respuesta += "RMGRP ERROR: ESTE USUARIO NO CUENTA CON LOS PERMISOS PARA REALIZAR ESTA ACCION"
	}

	return respuesta
}
