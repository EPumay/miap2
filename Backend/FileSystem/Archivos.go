package FileSystem

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"proyecto1/Structs"
	"proyecto1/Utilities"
	"strconv"
	"strings"
	"time"
)

func Mkdir(entrada []string) string {
	respuesta := "Comando mkdir"
	var path string
	p := false
	UsuarioA := Structs.UsuarioActual

	if !UsuarioA.Status {
		fmt.Println("ERROR MKFILE: SESION NO INICIADA")
		respuesta += "ERROR MKFILE: NO HAY SECION INICIADA" + "\n"
		respuesta += "POR FAVOR INICIAR SESION PARA CONTINUAR" + "\n"
		return respuesta
	}

	for _, parametro := range entrada[1:] {
		tmp := strings.TrimRight(parametro, " ")
		valores := strings.Split(tmp, "=")

		if strings.ToLower(valores[0]) == "path" {
			if len(valores) != 2 {
				fmt.Println("ERROR MKDIR, valor desconocido de parametros ", valores[1])
				respuesta += "ERROR MKDIR, valor desconocido de parametros " + valores[1]
				//Si falta el valor del parametro actual lo reconoce como error e interrumpe el proceso
				return respuesta
			}
			path = strings.ReplaceAll(valores[1], "\"", "")
		} else if strings.ToLower(valores[0]) == "p" {
			if len(tmp) != 1 {
				fmt.Println("MKDIR Error: Valor desconocido del parametro ", valores[0])
				return "MKDIR Error: Valor desconocido del parametro " + valores[0]
			}
			p = true

			//ERROR
		} else {
			fmt.Println("MKFILE ERROR: Parametro desconocido: ", valores[0])
			return "MKFILE ERROR: Parametro desconocido: " + valores[0]
		}
	}

	if path == "" {
		fmt.Println("MKDIR ERROR NO SE INGRESO PARAMETRO PATH")
		return "MKDIR ERROR NO SE INGRESO PARAMETRO PATH"
	}

	//Abrimos el disco
	Disco, err := Utilities.OpenFile(UsuarioA.DiskPath)
	if err != nil {
		return "MKFILE ERROR OPEN FILE " + err.Error() + "\n"
	}

	var mbr Structs.MBR
	// Read object from bin file
	if err := Utilities.ReadObject(Disco, &mbr, 0); err != nil {
		return "MKFILE ERROR READ FILE " + err.Error() + "\n"
	}

	// Close bin file
	defer Disco.Close()

	//Encontrar la particion correcta
	agregar := false
	part := -1 //particion a utilizar y modificar
	for i := 0; i < 4; i++ {
		identificador := Structs.GetId(string(mbr.Partitions[i].Id[:]))
		if identificador == UsuarioA.IdPart {
			part = i
			agregar = true
			break //para que ya no siga recorriendo si ya encontro la particion independientemente si se pudo o no reducir
		}
	}

	if agregar {
		var superBloque Structs.Superblock
		errREAD := Utilities.ReadObject(Disco, &superBloque, int64(mbr.Partitions[part].Start))
		if errREAD != nil {
			fmt.Println("MKFILE ERROR. Particion sin formato")
			return "MKFILE ERROR. Particion sin formato" + "\n"
		}

		//Validar que exista la ruta
		stepPath := strings.Split(path, "/")
		idInicial := int32(0)
		idActual := int32(0)
		crear := -1
		for i, itemPath := range stepPath[1:] {
			idActual = BuscarInodo(idInicial, "/"+itemPath, superBloque, Disco)
			if idInicial != idActual {
				idInicial = idActual
			} else {
				crear = i + 1 //porque estoy iniciando desde 1 e i inicia en 0
				break
			}
		}

		//crear carpetas padre si se tiene permiso
		if crear != -1 {
			if crear == len(stepPath)-1 {
				CreaCarpeta(idInicial, stepPath[crear], int64(mbr.Partitions[part].Start), Disco)
			} else {
				if p {
					for _, item := range stepPath[crear:] {
						idInicial = CreaCarpeta(idInicial, item, int64(mbr.Partitions[part].Start), Disco)
						if idInicial == 0 {
							fmt.Println("MKDIR ERROR: No se pudo crear carpeta")
							return "MKDIR ERROR: No se pudo crear carpeta"
						}
					}
				} else {
					fmt.Println("MKDIR ERROR: Sin permiso de crear carpetas padre")
				}
			}
			return "Carpeta(s) creada"
		} else {
			fmt.Println("MKDIR ERROR: LA CARPETA YA EXISTE")
			return "MKDIR ERROR: LA CARPETA YA EXISTE"
		}
	}
	return respuesta
}

func BuscarInodo(idInodo int32, path string, superBloque Structs.Superblock, file *os.File) int32 {
	//Dividir la ruta por cada /
	stepsPath := strings.Split(path, "/")
	//el arreglo vendra [ ,val1, val2] por lo que me corro una posicion
	tmpPath := stepsPath[1:]
	//fmt.Println("Ruta actual ", tmpPath)

	//cargo el inodo a partir del cual voy a buscar
	var Inode0 Structs.Inode
	Utilities.ReadObject(file, &Inode0, int64(superBloque.S_inode_start+(idInodo*int32(binary.Size(Structs.Inode{})))))
	//Recorrer los bloques directos (carpetas/archivos) en la raiz
	var folderBlock Structs.Folderblock
	for i := 0; i < 12; i++ {
		idBloque := Inode0.I_block[i]
		if idBloque != -1 {
			Utilities.ReadObject(file, &folderBlock, int64(superBloque.S_block_start+(idBloque*int32(binary.Size(Structs.Folderblock{})))))
			//Recorrer el bloque actual buscando la carpeta/archivo en la raiz
			for j := 2; j < 4; j++ {
				//apuntador es el apuntador del bloque al inodo (carpeta/archivo), si existe es distinto a -1
				apuntador := folderBlock.B_content[j].B_inodo
				if apuntador != -1 {
					pathActual := Structs.GetB_name(string(folderBlock.B_content[j].B_name[:]))
					if tmpPath[0] == pathActual {
						//buscarInodo(apuntador, ruta[1:], path, superBloque, iSuperBloque, file, r)
						if len(tmpPath) > 1 {
							return buscarIrecursivo(apuntador, tmpPath[1:], superBloque.S_inode_start, superBloque.S_block_start, file)
						} else {
							return apuntador
						}
					}
				}
			}
		}
	}
	//agregar busqueda en los apuntadores indirectos
	//i=12 -> simple; i=13 -> doble; i=14 -> triple
	//Si no encontro nada retornar 0 (la raiz)
	return idInodo
}

// Buscar inodo de forma recursiva
func buscarIrecursivo(idInodo int32, path []string, iStart int32, bStart int32, file *os.File) int32 {
	//cargo el inodo actual
	var inodo Structs.Inode
	Utilities.ReadObject(file, &inodo, int64(iStart+(idInodo*int32(binary.Size(Structs.Inode{})))))

	//Nota: el inodo tiene tipo. No es necesario pero se podria validar que sea carpeta
	//recorro el inodo buscando la siguiente carpeta
	var folderBlock Structs.Folderblock
	for i := 0; i < 12; i++ {
		idBloque := inodo.I_block[i]
		if idBloque != -1 {
			Utilities.ReadObject(file, &folderBlock, int64(bStart+(idBloque*int32(binary.Size(Structs.Folderblock{})))))
			//Recorrer el bloque buscando la carpeta actua
			for j := 2; j < 4; j++ {
				apuntador := folderBlock.B_content[j].B_inodo
				if apuntador != -1 {
					pathActual := Structs.GetB_name(string(folderBlock.B_content[j].B_name[:]))
					if path[0] == pathActual {
						if len(path) > 1 {
							//sin este if path[1:] termina en un arreglo de tamaño 0 y retornaria -1
							return buscarIrecursivo(apuntador, path[1:], iStart, bStart, file)
						} else {
							//cuando el arreglo path tiene tamaño 1 esta en la carpeta que busca
							return apuntador
						}
					}
				}
			}
		}
	}
	//agregar busqueda en los apuntadores indirectos
	//i=12 -> simple; i=13 -> doble; i=14 -> triple
	return -1
}

func CreaCarpeta(idInode int32, carpeta string, initSuperBloque int64, disco *os.File) int32 {
	var superBloque Structs.Superblock
	Utilities.ReadObject(disco, &superBloque, initSuperBloque)

	var inodo Structs.Inode
	Utilities.ReadObject(disco, &inodo, int64(superBloque.S_inode_start+(idInode*int32(binary.Size(Structs.Inode{})))))

	//Recorrer los bloques directos del inodo para ver si hay espacio libre
	for i := 0; i < 12; i++ {
		idBloque := inodo.I_block[i]
		if idBloque != -1 {
			//Existe un folderblock con idBloque que se debe revisar si tiene espacio para la nueva carpeta
			var folderBlock Structs.Folderblock
			Utilities.ReadObject(disco, &folderBlock, int64(superBloque.S_block_start+(idBloque*int32(binary.Size(Structs.Folderblock{})))))

			//Recorrer el bloque para ver si hay espacio
			for j := 2; j < 4; j++ {
				apuntador := folderBlock.B_content[j].B_inodo
				//Hay espacio en el bloque
				if apuntador == -1 {
					//modifico el bloque actual
					copy(folderBlock.B_content[j].B_name[:], carpeta)
					ino := superBloque.S_first_ino //primer inodo libre
					folderBlock.B_content[j].B_inodo = ino
					//ACTUALIZAR EL FOLDERBLOCK ACTUAL (idBloque) EN EL ARCHIVO
					Utilities.WriteObject(disco, folderBlock, int64(superBloque.S_block_start+(idBloque*int32(binary.Size(Structs.Folderblock{})))))

					//creo el nuevo inodo /ruta
					var newInodo Structs.Inode
					newInodo.I_uid = Structs.UsuarioActual.IdUsr
					newInodo.I_gid = Structs.UsuarioActual.IdGrp
					newInodo.I_size = 0 //es carpeta
					//Agrego las fechas
					ahora := time.Now()
					date := ahora.Format("02/01/2006 15:04")
					copy(newInodo.I_atime[:], date)
					copy(newInodo.I_ctime[:], date)
					copy(newInodo.I_mtime[:], date)
					copy(newInodo.I_type[:], "0") //es carpeta
					copy(newInodo.I_mtime[:], "664")

					//apuntadores iniciales
					for i := int32(0); i < 15; i++ {
						newInodo.I_block[i] = -1
					}
					//El apuntador a su primer bloque (el primero disponible)
					block := superBloque.S_first_blo
					newInodo.I_block[0] = block
					//escribo el nuevo inodo (ino)
					Utilities.WriteObject(disco, newInodo, int64(superBloque.S_inode_start+(ino*int32(binary.Size(Structs.Inode{})))))

					//crear el nuevo bloque
					var newFolderBlock Structs.Folderblock
					newFolderBlock.B_content[0].B_inodo = ino //idInodo actual
					copy(newFolderBlock.B_content[0].B_name[:], ".")
					newFolderBlock.B_content[1].B_inodo = folderBlock.B_content[0].B_inodo //el padre es el bloque anterior
					copy(newFolderBlock.B_content[1].B_name[:], "..")
					newFolderBlock.B_content[2].B_inodo = -1
					newFolderBlock.B_content[3].B_inodo = -1
					//escribo el nuevo bloque (block)
					Utilities.WriteObject(disco, newFolderBlock, int64(superBloque.S_block_start+(block*int32(binary.Size(Structs.Folderblock{})))))

					//modifico el superbloque
					superBloque.S_free_inodes_count -= 1
					superBloque.S_free_blocks_count -= 1
					superBloque.S_first_blo += 1
					superBloque.S_first_ino += 1
					//Escribir en el archivo los cambios del superBloque
					Utilities.WriteObject(disco, superBloque, initSuperBloque)

					//escribir el bitmap de bloques (se uso un bloque).
					Utilities.WriteObject(disco, byte(1), int64(superBloque.S_bm_block_start+block))

					//escribir el bitmap de inodos (se uso un inodo).
					Utilities.WriteObject(disco, byte(1), int64(superBloque.S_bm_inode_start+ino))
					//retorna el inodo creado (por si va a crear otra carpeta en ese inodo)
					return ino
				}
			} //fin de for de buscar espacio en el bloque actual (existente)
			//Fin if idBLoque existente
		} else {
			//No hay bloques con espacio disponible (existe al menos el primer bloque pero esta lleno)
			//modificar el inodo actual (por el nuevo apuntador)
			block := superBloque.S_first_blo //primer bloque libre
			inodo.I_block[i] = block
			//Escribir los cambios del inodo inicial
			Utilities.WriteObject(disco, &inodo, int64(superBloque.S_inode_start+(idInode*int32(binary.Size(Structs.Inode{})))))

			//cargo el primer bloque del inodo actual para tomar los datos de actual y padre (son los mismos para el nuevo)
			var folderBlock Structs.Folderblock
			bloque := inodo.I_block[0] //cargo el primer folderblock para obtener los datos del actual y su padre
			Utilities.ReadObject(disco, &folderBlock, int64(superBloque.S_block_start+(bloque*int32(binary.Size(Structs.Folderblock{})))))

			//creo el bloque que va a apuntar a la nueva carpeta
			var newFolderBlock1 Structs.Folderblock
			newFolderBlock1.B_content[0].B_inodo = folderBlock.B_content[0].B_inodo //actual
			copy(newFolderBlock1.B_content[0].B_name[:], ".")
			newFolderBlock1.B_content[1].B_inodo = folderBlock.B_content[1].B_inodo //padre
			copy(newFolderBlock1.B_content[1].B_name[:], "..")
			ino := superBloque.S_first_ino                        //primer inodo libre
			newFolderBlock1.B_content[2].B_inodo = ino            //apuntador al inodo nuevo
			copy(newFolderBlock1.B_content[2].B_name[:], carpeta) //nombre del inodo nuevo
			newFolderBlock1.B_content[3].B_inodo = -1
			//escribo el nuevo bloque (block)
			Utilities.WriteObject(disco, newFolderBlock1, int64(superBloque.S_block_start+(block*int32(binary.Size(Structs.Folderblock{})))))

			//creo el nuevo inodo /ruta
			var newInodo Structs.Inode
			newInodo.I_uid = Structs.UsuarioActual.IdUsr
			newInodo.I_gid = Structs.UsuarioActual.IdGrp
			newInodo.I_size = 0 //es carpeta
			//Agrego las fechas
			ahora := time.Now()
			date := ahora.Format("02/01/2006 15:04")
			copy(newInodo.I_atime[:], date)
			copy(newInodo.I_ctime[:], date)
			copy(newInodo.I_mtime[:], date)
			copy(newInodo.I_type[:], "0") //es carpeta
			copy(newInodo.I_mtime[:], "664")

			//apuntadores iniciales
			for i := int32(0); i < 15; i++ {
				newInodo.I_block[i] = -1
			}
			//El apuntador a su primer bloque (el primero disponible)
			block2 := superBloque.S_first_blo + 1
			newInodo.I_block[0] = block2
			//escribo el nuevo inodo (ino) creado en newFolderBlock1
			Utilities.WriteObject(disco, newInodo, int64(superBloque.S_inode_start+(ino*int32(binary.Size(Structs.Inode{})))))

			//crear nuevo bloque del inodo
			var newFolderBlock2 Structs.Folderblock
			newFolderBlock2.B_content[0].B_inodo = ino //idInodo actual
			copy(newFolderBlock2.B_content[0].B_name[:], ".")
			newFolderBlock2.B_content[1].B_inodo = newFolderBlock1.B_content[0].B_inodo //el padre es el bloque anterior
			copy(newFolderBlock2.B_content[1].B_name[:], "..")
			newFolderBlock2.B_content[2].B_inodo = -1
			newFolderBlock2.B_content[3].B_inodo = -1
			//escribo el nuevo bloque
			Utilities.WriteObject(disco, newFolderBlock2, int64(superBloque.S_block_start+(block2*int32(binary.Size(Structs.Folderblock{})))))

			//modifico el superbloque
			superBloque.S_free_inodes_count -= 1
			superBloque.S_free_blocks_count -= 2
			superBloque.S_first_blo += 2
			superBloque.S_first_ino += 1
			Utilities.WriteObject(disco, superBloque, initSuperBloque)

			//escribir el bitmap de bloques (se uso dos bloques: block y block2).
			Utilities.WriteObject(disco, byte(1), int64(superBloque.S_bm_block_start+block))
			Utilities.WriteObject(disco, byte(1), int64(superBloque.S_bm_block_start+block2))

			//escribir el bitmap de inodos (se uso un inodo: ino).
			Utilities.WriteObject(disco, byte(1), int64(superBloque.S_bm_inode_start+ino))
			return ino
		}
	} // Fin for bloques directos
	return 0
}

func Mkfile(entrada []string) string {
	respuesta := "Comando mkfile"
	parametrosDesconocidos := false
	var path string
	var cont string //path del archivo que esta en nuestra maquina y se copiara en el usuario utilizado
	size := 0       //opcional, si no viene toma valor 0
	r := false
	UsuarioA := Structs.UsuarioActual
	fmt.Println("dentro de mk file imprimir", entrada)
	if !UsuarioA.Status {
		fmt.Println("ERROR MKFILE: SESION NO INICIADA")
		respuesta += "ERROR MKFILE: NO HAY SECION INICIADA" + "\n"
		respuesta += "POR FAVOR INICIAR SESION PARA CONTINUAR" + "\n"
		return respuesta
	}

	for _, parametro := range entrada[1:] {
		tmp := strings.TrimRight(parametro, " ")
		valores := strings.Split(tmp, "=")

		if len(valores) == 2 {
			// --------------- PAHT ------------------
			if strings.ToLower(valores[0]) == "path" {
				path = strings.ReplaceAll(valores[1], "\"", "")
				fmt.Println("ahhhhhhh", path)
				//-------------- SIZE ---------------------
			} else if strings.ToLower(valores[0]) == "size" {
				//convierto a tipo int
				var err error
				size, err = strconv.Atoi(valores[1]) //se convierte el valor en un entero
				if err != nil {
					fmt.Println("MKFILE Error: Size solo acepta valores enteros. Ingreso: ", valores[1])
					return "MKFILE Error: Size solo acepta valores enteros. Ingreso: " + valores[1]
				}

				//valido que sea mayor a 0
				if size < 0 {
					fmt.Println("MKFILE Error: Size solo acepta valores positivos. Ingreso: ", valores[1])
					return "MKFILE Error: Size solo acepta valores positivos. Ingreso: " + valores[1]
				}
				//-------------- CONT ---------------------
			} else if strings.ToLower(valores[0]) == "cont" {
				cont = strings.ReplaceAll(valores[1], "\"", "")
				_, err := os.Stat(cont)
				if os.IsNotExist(err) {
					fmt.Println("MKFILE Error: El archivo cont no existe")
					respuesta += "MKFILE Error: El archivo cont no existe" + "\n"
					return respuesta // Terminar el bucle porque encontramos un nombre único
				}
			} else {
				parametrosDesconocidos = true
			}
		} else if len(valores) == 1 {
			if strings.ToLower(valores[0]) == "r" {
				r = true
			} else {
				parametrosDesconocidos = true
			}
		} else {
			parametrosDesconocidos = true
		}

		if parametrosDesconocidos {
			fmt.Println("MKFILE Error: Parametro desconocido: ", valores[0])
			respuesta += "MKFILE Error: Parametro desconocido: " + valores[0]
			return respuesta //por si en el camino reconoce algo invalido de una vez se sale
		}
	}

	if path == "" {
		fmt.Println("MKFIEL ERROR NO SE INGRESO PARAMETRO PATH")
		return "MKFIEL ERROR NO SE INGRESO PARAMETRO PATH"
	}

	//Abrimos el disco
	Disco, err := Utilities.OpenFile(UsuarioA.DiskPath)
	if err != nil {
		return "MKFILE ERROR OPEN FILE " + err.Error() + "\n"
	}

	var mbr Structs.MBR
	// Read object from bin file
	if err := Utilities.ReadObject(Disco, &mbr, 0); err != nil {
		return "MKFILE ERROR READ FILE " + err.Error() + "\n"
	}

	// Close bin file
	defer Disco.Close()

	//Encontrar la particion correcta
	agregar := false
	part := -1 //particion a utilizar y modificar
	for i := 0; i < 4; i++ {
		identificador := Structs.GetId(string(mbr.Partitions[i].Id[:]))
		if identificador == UsuarioA.IdPart {
			part = i
			agregar = true
			break //para que ya no siga recorriendo si ya encontro la particion independientemente si se pudo o no reducir
		}
	}

	if agregar {
		var superBloque Structs.Superblock
		errREAD := Utilities.ReadObject(Disco, &superBloque, int64(mbr.Partitions[part].Start))
		if errREAD != nil {
			fmt.Println("MKFILE ERROR. Particion sin formato")
			return "MKFILE ERROR. Particion sin formato" + "\n"
		}

		//Validar que exista la ruta
		stepPath := strings.Split(path, "/")
		finRuta := len(stepPath) - 1 //es el archivo -> stepPath[finRuta] = archivoNuevo.txt
		idInicial := int32(0)
		idActual := int32(0)
		crear := -1
		//No incluye a finRuta, es decir, se queda en el aterior. EJ: Tamaño=5, finRuta=4. El ultimo que evalua es stepPath[3]
		for i, itemPath := range stepPath[1:finRuta] {
			idActual = BuscarInodo(idInicial, "/"+itemPath, superBloque, Disco)
			//si el actual y el inicial son iguales significa que no existe la carpeta
			if idInicial != idActual {
				idInicial = idActual
			} else {
				crear = i + 1 //porque estoy iniciando desde 1 e i inicia en 0
				break
			}
		}

		//crear carpetas padre si se tiene permiso
		if crear != -1 {
			if r {
				for _, item := range stepPath[crear:finRuta] {
					idInicial = CreaCarpeta(idInicial, item, int64(mbr.Partitions[part].Start), Disco)
					if idInicial == 0 {
						fmt.Println("MKDIR ERROR: No se pudo crear carpeta")
						return "MKFILE ERROR: No se pudo crear carpeta"
					}
				}
			} else {
				fmt.Println("MKDIR ERROR: Carpeta ", stepPath[crear], " no existe. Sin permiso de crear carpetas padre")
				return "MKFILE ERROR: Carpeta " + stepPath[crear] + " no existe. Sin permiso de crear carpetas padre"
			}

		}

		//verificar que no exista el archivo (recordar que BuscarInodo busca de la forma /nombreBuscar)
		idNuevo := BuscarInodo(idInicial, "/"+stepPath[finRuta], superBloque, Disco)
		if idNuevo == idInicial {
			if cont == "" {
				digito := 0
				var content string

				//Crea el contenido del archivo con digitos del 0 al 9
				for i := 0; i < size; i++ {
					if digito == 10 {
						digito = 0
					}
					content += strconv.Itoa(digito)
					digito++
				}
				respuesta = crearArchivo(idInicial, stepPath[finRuta], size, content, int64(mbr.Partitions[part].Start), Disco)
			} else {
				archivoC, err := Utilities.OpenFile(cont)
				if err != nil {
					return "MKFILE ERROR OPEN FILE " + err.Error() + "\n"
				}

				//lee el contenido del archivo
				content, err := ioutil.ReadFile(cont)
				if err != nil {
					fmt.Println(err)
					return "ERROR MKFILE " + err.Error()
				}
				// Close bin file
				defer archivoC.Close()
				respuesta = crearArchivo(idInicial, stepPath[finRuta], size, string(content), int64(mbr.Partitions[part].Start), Disco)
			}
		} else {
			fmt.Println("El archivo ya existe")
			return "ERROR: El archivo ya existe"
		}
	}
	return respuesta
}

func crearArchivo(idInodo int32, file string, size int, contenido string, initSuperBloque int64, disco *os.File) string {
	//cargar el superBloque actual
	var superB Structs.Superblock
	Utilities.ReadObject(disco, &superB, initSuperBloque)
	// cargo el inodo de la carpeta que contendra el archivo
	var inodoFile Structs.Inode
	Utilities.ReadObject(disco, &inodoFile, int64(superB.S_inode_start+(idInodo*int32(binary.Size(Structs.Inode{})))))

	//recorro el inodo de la carpeta para ver donde guardar el archivo (si hay espacio)
	for i := 0; i < 12; i++ {
		idBloque := inodoFile.I_block[i]
		if idBloque != -1 {
			//Existe un folderblock con idBloque que se debe revisar si tiene espacio para el nuevo archivo
			var folderBlock Structs.Folderblock
			Utilities.ReadObject(disco, &folderBlock, int64(superB.S_block_start+(idBloque*int32(binary.Size(Structs.Folderblock{})))))

			//Recorrer el bloque para ver si hay espacio y si hay crear el archivo
			for j := 2; j < 4; j++ {
				apuntador := folderBlock.B_content[j].B_inodo
				//Hay espacio en el bloque
				if apuntador == -1 {
					//modifico el bloque actual
					copy(folderBlock.B_content[j].B_name[:], file)
					ino := superB.S_first_ino //primer inodo libre
					folderBlock.B_content[j].B_inodo = ino
					//ACTUALIZAR EL FOLDERBLOCK ACTUAL (idBloque) EN EL ARCHIVO
					Utilities.WriteObject(disco, folderBlock, int64(superB.S_block_start+(idBloque*int32(binary.Size(Structs.Folderblock{})))))

					//creo el nuevo inodo archivo
					var newInodo Structs.Inode
					newInodo.I_uid = Structs.UsuarioActual.IdUsr
					newInodo.I_gid = Structs.UsuarioActual.IdGrp
					newInodo.I_size = int32(size) //Size es el tamaño del archivo
					//Agrego las fechas
					ahora := time.Now()
					date := ahora.Format("02/01/2006 15:04")
					copy(newInodo.I_atime[:], date)
					copy(newInodo.I_ctime[:], date)
					copy(newInodo.I_mtime[:], date)
					copy(newInodo.I_type[:], "1") //es archivo
					copy(newInodo.I_perm[:], "664")

					//apuntadores iniciales
					for i := int32(0); i < 15; i++ {
						newInodo.I_block[i] = -1
					}

					//El apuntador a su primer bloque (el primero disponible)
					fileblock := superB.S_first_blo

					//division del contenido en los fileblocks de 64 bytes
					inicio := 0
					fin := 0
					sizeContenido := len(contenido)
					if sizeContenido < 64 {
						fin = len(contenido)
					} else {
						fin = 64
					}

					//crear el/los fileblocks con el contenido del archivo0
					for i := int32(0); i < 12; i++ {
						newInodo.I_block[i] = fileblock
						//Guardar la informacion del bloque
						data := contenido[inicio:fin]
						var newFileBlock Structs.Fileblock
						copy(newFileBlock.B_content[:], []byte(data))
						//escribo el nuevo bloque (fileblock)
						Utilities.WriteObject(disco, newFileBlock, int64(superB.S_block_start+(fileblock*int32(binary.Size(Structs.Fileblock{})))))

						//modifico el superbloque (solo el bloque usado por iteracion)
						superB.S_free_blocks_count -= 1
						superB.S_first_blo += 1

						//escribir el bitmap de bloques (se usa un bloque por iteracion).
						Utilities.WriteObject(disco, byte(1), int64(superB.S_bm_block_start+fileblock))

						//validar si queda data que agregar al archivo para continuar con el ciclo o detenerlo
						calculo := len(contenido[fin:])
						if calculo > 64 {
							inicio = fin
							fin += 64
						} else if calculo > 0 {
							inicio = fin
							fin += calculo
						} else {
							//detener el ciclo de creacion de fileblocks
							break
						}
						//Aumento el fileblock
						fileblock++
					}

					//escribo el nuevo inodo (ino)
					Utilities.WriteObject(disco, newInodo, int64(superB.S_inode_start+(ino*int32(binary.Size(Structs.Inode{})))))

					//modifico el superbloque por el inodo usado
					superB.S_free_inodes_count -= 1
					superB.S_first_ino += 1
					//Escribir en el archivo los cambios del superBloque
					Utilities.WriteObject(disco, superB, initSuperBloque)

					//escribir el bitmap de inodos (se uso un inodo).
					Utilities.WriteObject(disco, byte(1), int64(superB.S_bm_inode_start+ino))

					return "Archivo creado exitosamente"
				} //Fin if apuntadores
			} //fin For bloques
		} else {
			//No hay bloques con espacio disponible
			//modificar el inodo actual (por el nuevo apuntador)
			block := superB.S_first_blo //primer bloque libre
			inodoFile.I_block[i] = block
			//Escribir los cambios del inodo inicial
			Utilities.WriteObject(disco, &inodoFile, int64(superB.S_inode_start+(idInodo*int32(binary.Size(Structs.Inode{})))))

			//cargo el primer bloque del inodo actual para tomar los datos de actual y padre (son los mismos para el nuevo)
			var folderBlock Structs.Folderblock
			bloque := inodoFile.I_block[0] //cargo el primer folderblock para obtener los datos del actual y su padre
			Utilities.ReadObject(disco, &folderBlock, int64(superB.S_block_start+(bloque*int32(binary.Size(Structs.Folderblock{})))))

			//creo el primer bloque que va a apuntar al nuevo archivo
			var newFolderBlock1 Structs.Folderblock
			newFolderBlock1.B_content[0].B_inodo = folderBlock.B_content[0].B_inodo //actual
			copy(newFolderBlock1.B_content[0].B_name[:], ".")
			newFolderBlock1.B_content[1].B_inodo = folderBlock.B_content[1].B_inodo //padre
			copy(newFolderBlock1.B_content[1].B_name[:], "..")
			ino := superB.S_first_ino                          //primer inodo libre
			newFolderBlock1.B_content[2].B_inodo = ino         //apuntador al inodo nuevo
			copy(newFolderBlock1.B_content[2].B_name[:], file) //nombre del inodo nuevo
			newFolderBlock1.B_content[3].B_inodo = -1
			//escribo el nuevo bloque (block)
			Utilities.WriteObject(disco, newFolderBlock1, int64(superB.S_block_start+(block*int32(binary.Size(Structs.Folderblock{})))))

			//escribir el bitmap de bloques
			Utilities.WriteObject(disco, byte(1), int64(superB.S_bm_block_start+block))

			//modifico el superbloque porque mas adelante lo necesito con estos cambios
			superB.S_first_blo += 1
			superB.S_free_blocks_count -= 1

			//creo el nuevo inodo archivo
			var newInodo Structs.Inode
			newInodo.I_uid = Structs.UsuarioActual.IdUsr
			newInodo.I_gid = Structs.UsuarioActual.IdGrp
			newInodo.I_size = int32(size) //Size es el tamaño del archivo
			//Agrego las fechas
			ahora := time.Now()
			date := ahora.Format("02/01/2006 15:04")
			copy(newInodo.I_atime[:], date)
			copy(newInodo.I_ctime[:], date)
			copy(newInodo.I_mtime[:], date)
			copy(newInodo.I_type[:], "1") //es archivo
			copy(newInodo.I_mtime[:], "664")

			//apuntadores iniciales
			for i := int32(0); i < 15; i++ {
				newInodo.I_block[i] = -1
			}

			//El apuntador a su primer bloque (el primero disponible)
			fileblock := superB.S_first_blo

			//division del contenido en los fileblocks de 64 bytes
			inicio := 0
			fin := 0
			sizeContenido := len(contenido)
			if sizeContenido < 64 {
				fin = len(contenido)
			} else {
				fin = 64
			}

			//crear el/los fileblocks con el contenido del archivo0
			for i := int32(0); i < 12; i++ {
				newInodo.I_block[i] = fileblock
				//Guardar la informacion del bloque
				data := contenido[inicio:fin]
				var newFileBlock Structs.Fileblock
				copy(newFileBlock.B_content[:], []byte(data))
				//escribo el nuevo bloque (fileblock)
				Utilities.WriteObject(disco, newFileBlock, int64(superB.S_block_start+(fileblock*int32(binary.Size(Structs.Fileblock{})))))

				//modifico el superbloque (solo el bloque usado por iteracion)
				superB.S_free_blocks_count -= 1
				superB.S_first_blo += 1

				//escribir el bitmap de bloques (se usa un bloque por iteracion).
				Utilities.WriteObject(disco, byte(1), int64(superB.S_bm_block_start+fileblock))

				//validar si queda data que agregar al archivo para continuar con el ciclo o detenerlo
				calculo := len(contenido[fin:])
				if calculo > 64 {
					inicio = fin
					fin += 64
				} else if calculo > 0 {
					inicio = fin
					fin += calculo
				} else {
					//detener el ciclo de creacion de fileblocks
					break
				}
				//Aumento el fileblock
				fileblock++
			}

			//escribo el nuevo inodo (ino)
			Utilities.WriteObject(disco, newInodo, int64(superB.S_inode_start+(ino*int32(binary.Size(Structs.Inode{})))))

			//modifico el superbloque por el inodo usado
			superB.S_free_inodes_count -= 1
			superB.S_first_ino += 1
			//Escribir en el archivo los cambios del superBloque
			Utilities.WriteObject(disco, superB, initSuperBloque)

			//escribir el bitmap de inodos (se uso un inodo).
			Utilities.WriteObject(disco, byte(1), int64(superB.S_bm_inode_start+ino))

			return "Archivo creado exitosamente"
		}
	}

	return "ERROR MKFILE: OCURRIO UN ERROR INESPERADO AL CREAR EL ARCHIVO"
}

func Cat(entrada []string) string {
	respuesta := ""
	var filen []string

	UsuarioA := Structs.UsuarioActual

	if !UsuarioA.Status {
		respuesta += "ERROR CAT: NO HAY SECION INICIADA" + "\n"
		respuesta += "POR FAVOR INICIAR SESION PARA CONTINUAR" + "\n"
		return respuesta
	}

	for _, parametro := range entrada[1:] {
		tmp := strings.TrimRight(parametro, " ")
		valores := strings.Split(tmp, "=")

		if len(valores) != 2 {
			fmt.Println("ERROR CAT, valor desconocido de parametros ", valores[1])
			respuesta += "ERROR CAT, valor desconocido de parametros " + valores[1] + "\n"
			//Si falta el valor del parametro actual lo reconoce como error e interrumpe el proceso
			return respuesta
		}
		fileN := valores[0][:4] //toma los primeros 4 caracteres de valores[0]

		//******************** File *****************
		if strings.ToLower(fileN) == "file" {
			numero := strings.Split(strings.ToLower(valores[0]), "file")
			_, errId := strconv.Atoi(numero[1])
			if errId != nil {
				fmt.Println("CAT ERROR: No se pudo obtener un numero de fichero")
				return "CAT ERROR: No se pudo obtener un numero de fichero"
			}
			//eliminar comillas
			tmp1 := strings.ReplaceAll(valores[1], "\"", "")
			filen = append(filen, tmp1)
			//******************* ERROR EN LOS PARAMETROS *************
		} else {
			fmt.Println("CAT ERROR: Parametro desconocido: ", valores[0])
			//por si en el camino reconoce algo invalido de una vez se sale
			return "CAT ERROR: Parametro desconocido: " + valores[0] + "\n"
		}
	}

	//Abrimos el disco
	Disco, err := Utilities.OpenFile(UsuarioA.DiskPath)
	if err != nil {
		return "CAR ERROR OPEN FILE " + err.Error() + "\n"
	}

	var mbr Structs.MBR
	// Read object from bin file
	if err := Utilities.ReadObject(Disco, &mbr, 0); err != nil {
		return "CAR ERROR READ FILE " + err.Error() + "\n"
	}

	// Close bin file
	defer Disco.Close()

	//Encontrar la particion correcta
	buscar := false
	part := -1 //particion a utilizar y modificar
	for i := 0; i < 4; i++ {
		identificador := Structs.GetId(string(mbr.Partitions[i].Id[:]))
		if identificador == UsuarioA.IdPart {
			part = i
			buscar = true
			break
		}
	}

	if buscar {
		var contenido string
		var fileBlock Structs.Fileblock
		var superBloque Structs.Superblock

		errREAD := Utilities.ReadObject(Disco, &superBloque, int64(mbr.Partitions[part].Start))
		if errREAD != nil {
			errorMsg := "CAT ERROR. Particion sin formato\n"
			fmt.Println(errorMsg)
			return errorMsg // Retorna directamente el mensaje de error
		}

		for _, item := range filen {
			idInodo := BuscarInodo(0, item, superBloque, Disco)
			var inodo Structs.Inode

			if idInodo > 0 {
				contenido += "\nContenido del archivo: '" + item + "':\n"
				fmt.Println("ID Inodo ", idInodo)
				Utilities.ReadObject(Disco, &inodo, int64(superBloque.S_inode_start+(idInodo*int32(binary.Size(Structs.Inode{})))))

				if inodo.I_uid == UsuarioA.IdUsr || UsuarioA.Nombre == "root" {
					for _, idBlock := range inodo.I_block {
						if idBlock != -1 {
							Utilities.ReadObject(Disco, &fileBlock, int64(superBloque.S_block_start+(idBlock*int32(binary.Size(Structs.Fileblock{})))))
							bloqueContenido := string(fileBlock.B_content[:])
							contenido += bloqueContenido // Agrega el contenido sin salto de línea extra
							fmt.Println("Bloque", idBlock, ":", bloqueContenido)
						}
					}
				} else {
					contenido += "ERROR CAT: No tiene permisos para visualizar el archivo " + item + "\n"
				}
			} else {
				contenido += "\nCAT ERROR: No se encontro el archivo " + item + "\n"
			}
		}

		respuesta = contenido // Asigna directamente el contenido a la respuesta
		fmt.Println("Contenido:", contenido)
	}

	return respuesta
}
