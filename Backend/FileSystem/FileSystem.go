package FileSystem

import (
	"encoding/binary"
	"fmt"
	"os"
	"proyecto1/Structs"
	"proyecto1/Utilities"
	"strings"
	"time"
)

func Mkfs(entrada []string) string {
	var respuesta string
	var id string //obligatorio
	fs := "2fs"   //2fs -> EXT2; 3fs -> EXT3; por defecto ext2
	Valido := true
	var pathDico string

	for _, parametro := range entrada[1:] {
		tmp := strings.TrimRight(parametro, " ")
		valores := strings.Split(tmp, "=")

		if len(valores) != 2 {
			fmt.Println("ERROR MKFS, valor desconocido de parametros ", valores[1])
			respuesta += "ERROR MKFS, valor desconocido de parametros " + valores[1]
			return respuesta
		}

		if strings.ToLower(valores[0]) == "id" {
			id = strings.ToUpper(valores[1])
		} else if strings.ToLower(valores[0]) == "type" {
			if strings.ToLower(valores[1]) != "full" {
				fmt.Println("MKFS Error. Valor de -type desconocido")
				respuesta += "MKFS Error. Valor de -type desconocido. "
				Valido = false
				return respuesta
			}
		} else if strings.ToLower(valores[0]) == "fs" {
			if strings.ToLower(valores[1]) == "3fs" {
				fs = "3fs"
			} else if strings.ToLower(valores[1]) != "2fs" {
				fmt.Println("MKFS Error en -fs. Valores aceptados: 2fs o 3fs. ingreso: ", valores[1])
				Valido = false
				break
			}

			//ERROR EN LOS PARAMETROS LEIDO
		} else {
			fmt.Println("MKFS Error: Parametro desconocido: ", valores[0])
			Valido = false
			return "MKFS Error: Parametro desconocido: " + valores[0] //por si en el camino reconoce algo invalido de una vez se sale
		}

	}

	if id != "" {
		//BUsca en struck de particiones montadas el id ingresado
		for _, montado := range Structs.Montadas {
			if montado.Id == id {
				pathDico = montado.PathM
			}
		}
		if pathDico == "" {
			respuesta += "ERROR MKFS ID INCORRECTO"
			fmt.Println("ERROR MKFS NO SE ENCONTRA EL ID")
			Valido = false
			return respuesta
		}
	} else {
		respuesta += "ERROR MKFS NO SE INGRESO ID"
		Valido = false
		return respuesta
	}

	if Valido {
		//Abrir el Disco
		Disco, err := Utilities.OpenFile(pathDico)
		if err != nil {
			respuesta += "ERROR MKFS MBR Open " + err.Error()
			return respuesta
		}

		//Crgar el mbr
		var mbr Structs.MBR
		// Read object from bin file
		if err := Utilities.ReadObject(Disco, &mbr, 0); err != nil {
			respuesta += "ERROR MKFS MBR Read " + err.Error()
			return respuesta
		}

		// Close bin file
		defer Disco.Close()

		//Buscar particion con el id solicitado
		formatear := true //usar si se van a montar logicas
		for i := 0; i < 4; i++ {
			identificador := Structs.GetId(string(mbr.Partitions[i].Id[:]))
			if identificador == id {
				formatear = false //Si encuentra la particion

				//Crear el super bloque que contiene los datos del sistema de archivos. Es similar al mbr en los discos
				var newSuperBloque Structs.Superblock
				Utilities.ReadObject(Disco, &newSuperBloque, int64(mbr.Partitions[i].Start))

				//Calcular el numero de inodos que caben en la particion. El numero de bloques es el triple de inodos
				//(formula a partir del tamaño de la particion, esta en el enunciado pag. 6)
				//tamaños fisicos: SuperBloque = 92; Inodo = 124; Bloque = 64

				numerador := int(mbr.Partitions[i].Size) - binary.Size(Structs.Superblock{})
				denominador := 4 + binary.Size(Structs.Inode{}) + 3*binary.Size(Structs.Fileblock{})
				if fs == "3fs" {
					denominador += binary.Size(Structs.Journaling{})
				}
				n := int32(numerador / denominador) //numero de inodos (tanto para EXT2 como para EXT3)

				//inicializar atributos generales del superbloque
				newSuperBloque.S_blocks_count = int32(3 * n)      //Total de bloques creados (pueden usarse)
				newSuperBloque.S_free_blocks_count = int32(3 * n) //Numero de bloques libre (Todos estan libres por ahora)

				newSuperBloque.S_inodes_count = n      //Total de inodos creados (pueden usarse)
				newSuperBloque.S_free_inodes_count = n //numero de inodos libres (todos estan libres por ahora)

				newSuperBloque.S_inode_size = int32(binary.Size(Structs.Inode{}))
				newSuperBloque.S_block_size = int32(binary.Size(Structs.Fileblock{}))

				//obtener hora de montaje del sistema de archivos
				ahora := time.Now()
				copy(newSuperBloque.S_mtime[:], ahora.Format("02/01/2006 15:04"))
				//Si fecha de desmontaje coincide con montaje es porque aun no se monta
				copy(newSuperBloque.S_umtime[:], ahora.Format("02/01/2006 15:04"))
				newSuperBloque.S_mnt_count += 1 //Se esta montando por primera vez
				newSuperBloque.S_magic = 0xEF53

				if fs == "2fs" {
					respuesta = crearEXT2(n, mbr.Partitions[i], newSuperBloque, ahora.Format("02/01/2006 15:04"), Disco)
				} else {
					respuesta = crearEXT3(n, mbr.Partitions[i], newSuperBloque, ahora.Format("02/01/2006 15:04"), Disco)
				}
				//Fin del formateo

				//Si hubiera una sesion iniciada eliminarla
				if Structs.UsuarioActual.Status {
					var new Structs.UserInfo
					Structs.UsuarioActual = new
				}
			}
		}

		if formatear {
			fmt.Println("MKFS Error. No se pudo formatear la particion con id ", id)
			fmt.Println("MKFS Error. No existe el id")
			respuesta += "MKFS Error. No se pudo formatear la particion con id " + id
			respuesta += "MKFS Error. No existe el id"
		}
	}

	return respuesta
}

func crearEXT2(n int32, particion Structs.Partition, newSuperBloque Structs.Superblock, date string, file *os.File) string {
	var respuesta string
	fmt.Println("Superbloque: ", newSuperBloque)
	fmt.Println("Fecha: ", date)

	//completar los atributos del super bloque. La estructura de la particion formateada es:
	// | Superbloque | Bitmap Inodos | Bitmap Bloques | Inodos | Bloques |

	//tipo del sistema de archivos
	newSuperBloque.S_filesystem_type = 2 //2 -> EXT2; 3 -> EXT3
	//Bitmap Inodos inicia donde termina el superbloque fisicamente (y el superbloque esta al inicio de la particion)
	newSuperBloque.S_bm_inode_start = particion.Start + int32(binary.Size(Structs.Superblock{}))
	//Bitmap bloques inicia donde termina el de inodos. Se suma n que es el numero de inodos maximo
	newSuperBloque.S_bm_block_start = newSuperBloque.S_bm_inode_start + n
	//Se crea el primer Inodo. Esta al final de los bloques que son 3 veces el numero de inodos
	newSuperBloque.S_inode_start = newSuperBloque.S_bm_block_start + 3*n
	//Se crea el primer bloque, este esta al final de los inodos fisicos
	newSuperBloque.S_block_start = newSuperBloque.S_inode_start + n*int32(binary.Size(Structs.Inode{}))

	//Se restan 2 bloques y dos inodos. uno para la carpeta raiz y otro para el archivo users.txt
	//lo que se crea al formatear es /users.txt (la carpeta usa un inodo y el archivo otro)
	newSuperBloque.S_free_inodes_count -= 2
	newSuperBloque.S_free_blocks_count -= 2

	//primer inodo libre
	//newSuperBloque.S_first_ino = newSuperBloque.S_inode_start + 2*int32(binary.Size(Structs.Inode{})) //multiplico por 2 porque hay 2 inodos creados
	newSuperBloque.S_first_ino = int32(2)
	//primer bloque libre
	//newSuperBloque.S_first_blo = newSuperBloque.S_block_start + 2*int32(binary.Size(Structs.Fileblock{})) //multiplicar por 2 porque hay 2 bloques creados
	newSuperBloque.S_first_blo = int32(2)

	//limpio (formateo) el espacio del bitmap de inodos para evitar inconsistencias
	bmInodeData := make([]byte, n)
	bmInodeErr := Utilities.WriteObject(file, bmInodeData, int64(newSuperBloque.S_bm_inode_start))
	if bmInodeErr != nil {
		fmt.Println("MKFS Error: ", bmInodeErr)
		respuesta += "MKFS Error: " + bmInodeErr.Error()
		return respuesta
	}

	//limpiar (formatear) el espacio del bitmap de bloques para evitar inconsistencias
	bmBlockData := make([]byte, 3*n)
	bmBlockErr := Utilities.WriteObject(file, bmBlockData, int64(newSuperBloque.S_bm_block_start))
	if bmBlockErr != nil {
		fmt.Println("MKFS Error: ", bmInodeErr)
		respuesta += "MKFS Error: " + bmBlockErr.Error()
		return respuesta
	}

	//creo un inodo y lleno el arreglo de bloques con -1
	var newInode Structs.Inode
	for i := 0; i < 15; i++ {
		newInode.I_block[i] = -1
	}

	//creo todos los inodos del sistema de archivos
	for i := int32(0); i < n; i++ {
		err := Utilities.WriteObject(file, newInode, int64(newSuperBloque.S_inode_start+i*int32(binary.Size(Structs.Inode{}))))
		if err != nil {
			fmt.Println("MKFS Error: ", err)
			return "MKFS Error: " + err.Error()
		}
	}

	//Crear todos los bloques de carpeta que se pueden crear
	fileBlocks := make([]Structs.Fileblock, 3*n) //lo puedo trabajar asi porque son instancias de la estructura, el inode llevaban valores
	fileBlocksErr := Utilities.WriteObject(file, fileBlocks, int64(newSuperBloque.S_bm_block_start))
	if fileBlocksErr != nil {
		fmt.Println("MKFS Error: ", fileBlocksErr)
		return "MKFS Error: " + fileBlocksErr.Error()
	}

	//Crear el Inode 0
	var Inode0 Structs.Inode
	Inode0.I_uid = 1
	Inode0.I_gid = 1
	Inode0.I_size = 0 //por ser carpeta no tiene tamaño como tal. para saber si existe basarse en I_ui/I_gid
	//unica vez que las 3 fechas son iguales
	copy(Inode0.I_atime[:], date)
	copy(Inode0.I_ctime[:], date)
	copy(Inode0.I_mtime[:], date)
	copy(Inode0.I_type[:], "0") //como es raiz es de tipo carpeta
	copy(Inode0.I_perm[:], "664")

	for i := int32(0); i < 15; i++ {
		Inode0.I_block[i] = -1
	}

	Inode0.I_block[0] = 0 //apunta al bloque 0

	var folderBlock0 Structs.Folderblock //Bloque0 -> carpetas
	folderBlock0.B_content[0].B_inodo = 0
	copy(folderBlock0.B_content[0].B_name[:], ".")
	folderBlock0.B_content[1].B_inodo = 0
	copy(folderBlock0.B_content[1].B_name[:], "..")
	folderBlock0.B_content[2].B_inodo = 1
	copy(folderBlock0.B_content[2].B_name[:], "users.txt")
	folderBlock0.B_content[3].B_inodo = -1

	//Inode1 que es el que contiene el archivo (Bloque 0 apunta a este nuevo inodo)
	var Inode1 Structs.Inode
	Inode1.I_uid = 1
	Inode1.I_gid = 1
	Inode1.I_size = int32(binary.Size(Structs.Folderblock{}))
	copy(Inode1.I_atime[:], date)
	copy(Inode1.I_ctime[:], date)
	copy(Inode1.I_mtime[:], date)
	copy(Inode1.I_type[:], "1") //es del archivo
	copy(Inode0.I_perm[:], "664")
	for i := int32(0); i < 15; i++ {
		Inode1.I_block[i] = -1
	}
	//Inode1 apunta al bloque1 (en este caso el bloque1 contiene el archivo)
	Inode1.I_block[0] = 1
	data := "1,G,root\n1,U,root,root,123\n"
	var fileBlock1 Structs.Fileblock //Bloque1 -> archivo
	copy(fileBlock1.B_content[:], []byte(data))
	fmt.Println("Creado users.txt con los datos : \n", data)

	//resumen
	//Inodo 0 -> Bloque 0 -> Inodo1 -> bloque1 (archivo)

	//Crear la carpeta raiz /
	//crear el archivo users.txt

	//fmt.Println("Superbloque: ", newSuperBloque)

	// Escribir el superbloque
	Utilities.WriteObject(file, newSuperBloque, int64(particion.Start))

	//escribir el bitmap de inodos
	Utilities.WriteObject(file, byte(1), int64(newSuperBloque.S_bm_inode_start))
	Utilities.WriteObject(file, byte(1), int64(newSuperBloque.S_bm_inode_start+1)) //Se escribieron dos inode

	//escribir el bitmap de bloques (se usaron dos bloques)
	Utilities.WriteObject(file, byte(1), int64(newSuperBloque.S_bm_block_start))
	Utilities.WriteObject(file, byte(1), int64(newSuperBloque.S_bm_block_start+1))

	//escribir inodes
	//Inode0
	Utilities.WriteObject(file, Inode0, int64(newSuperBloque.S_inode_start))
	//Inode1
	Utilities.WriteObject(file, Inode1, int64(newSuperBloque.S_inode_start+int32(binary.Size(Structs.Inode{}))))

	//Escribir bloques
	//bloque0
	Utilities.WriteObject(file, folderBlock0, int64(newSuperBloque.S_block_start))
	//bloque1
	Utilities.WriteObject(file, fileBlock1, int64(newSuperBloque.S_block_start+int32(binary.Size(Structs.Fileblock{}))))
	// Fin crear EXT2

	fmt.Println("La particion con nombre " + string(particion.Name[:]) + " fue formateada con exito con EXT2")
	return "La particion con nombre " + string(particion.Name[:]) + " fue formateada con exito con EXT2"
}

func crearEXT3(n int32, particion Structs.Partition, newSuperBloque Structs.Superblock, date string, file *os.File) string {
	fmt.Println("====== Crear EXT3 ======")
	fmt.Println("N: ", n)
	//fmt.Println("Superbloque: ", newSuperBloque)
	//fmt.Println("Fecha: ", date)
	fmt.Println("Journaling: ", binary.Size(Structs.Journaling{}))
	//completar los atributos del super bloque. La estructura de la particion formateada es:
	// | Superbloque | Journaling | Bitmap Inodos | Bitmap Bloques | Inodos | Bloques |

	//tipo del sistema de archivos
	newSuperBloque.S_filesystem_type = 3 //2 -> EXT2; 3 -> EXT3
	//Bitmap Inodos inicia donde termina el superbloque fisicamente (y el superbloque esta al inicio de la particion)
	newSuperBloque.S_bm_inode_start = particion.Start + int32(binary.Size(Structs.Superblock{})) + int32(binary.Size(Structs.Journaling{}))
	//Bitmap bloques inicia donde termina el de inodos. Se suma n que es el numero de inodos maximo
	newSuperBloque.S_bm_block_start = newSuperBloque.S_bm_inode_start + n
	//Se crea el primer Inodo. Esta al final de los bloques que son 3 veces el numero de inodos
	newSuperBloque.S_inode_start = newSuperBloque.S_bm_block_start + 3*n
	//Se crea el primer bloque, este esta al final de los inodos fisicos
	newSuperBloque.S_block_start = newSuperBloque.S_inode_start + n*int32(binary.Size(Structs.Inode{}))

	//Se restan 2 bloques y dos inodos. uno para la carpeta raiz y otro para el archivo users.txt
	//lo que se crea al formatear es /users.txt (la carpeta usa un inodo y el archivo otro)
	newSuperBloque.S_free_inodes_count -= 2
	newSuperBloque.S_free_blocks_count -= 2

	//primer inodo libre
	//newSuperBloque.S_first_ino = newSuperBloque.S_inode_start + 2*int32(binary.Size(Structs.Inode{})) //multiplico por 2 porque hay 2 inodos creados
	newSuperBloque.S_first_ino = int32(2)
	//primer bloque libre
	//newSuperBloque.S_first_blo = newSuperBloque.S_block_start + 2*int32(binary.Size(Structs.Fileblock{})) //multiplicar por 2 porque hay 2 bloques creados
	newSuperBloque.S_first_blo = int32(2)

	//Creo el primer registro del journaling. Este sera la creacion de la raiz
	var newJournal Structs.Journaling
	newJournal.Ultimo = 0
	newJournal.Size = int32(binary.Size(Structs.Journaling{}))

	//Creo el primer registro en la primer posicion del journal
	dataJ := newJournal.Contenido[0]
	copy(dataJ.Operation[:], "mkdir")
	copy(dataJ.Path[:], "/")
	copy(dataJ.Content[:], "-")
	copy(dataJ.Date[:], date)
	newJournal.Contenido[0] = dataJ

	//limpio (formateo) el espacio del bitmap de inodos para evitar inconsistencias
	bmInodeData := make([]byte, n)
	bmInodeErr := Utilities.WriteObject(file, bmInodeData, int64(newSuperBloque.S_bm_inode_start))
	if bmInodeErr != nil {
		fmt.Println("MKFS Error: ", bmInodeErr)
		return "MKFS Error: " + bmInodeErr.Error()
	}

	//limpiar (formatear) el espacio del bitmap de bloques para evitar inconsistencias
	bmBlockData := make([]byte, 3*n)
	bmBlockErr := Utilities.WriteObject(file, bmBlockData, int64(newSuperBloque.S_bm_block_start))
	if bmBlockErr != nil {
		fmt.Println("MKFS Error: ", bmBlockErr)
		return "MKFS Error: " + bmBlockErr.Error()
	}

	//creo un inodo y lleno el arreglo de bloques con -1
	var newInode Structs.Inode
	for i := 0; i < 15; i++ {
		newInode.I_block[i] = -1
	}

	//creo todos los inodos del sistema de archivos
	for i := int32(0); i < n; i++ {
		err := Utilities.WriteObject(file, newInode, int64(newSuperBloque.S_inode_start+i*int32(binary.Size(Structs.Inode{}))))
		if err != nil {
			fmt.Println("MKFS Error: ", err)
			return "MKFS Error: " + err.Error()
		}
	}

	//Crear todos los bloques de carpeta que se pueden crear
	fileBlocks := make([]Structs.Fileblock, 3*n) //lo puedo trabajar asi porque son instancias de la estructura, el inode llevaban valores
	fileBlocksErr := Utilities.WriteObject(file, fileBlocks, int64(newSuperBloque.S_bm_block_start))
	if fileBlocksErr != nil {
		fmt.Println("MKFS Error: ", fileBlocksErr)
		return "MKFS Error: " + fileBlocksErr.Error()
	}

	//Crear el Inode 0
	var Inode0 Structs.Inode
	Inode0.I_uid = 1
	Inode0.I_gid = 1
	Inode0.I_size = 0 //por ser carpeta no tiene tamaño como tal. para saber si existe basarse en I_ui/I_gid
	//unica vez que las 3 fechas son iguales
	copy(Inode0.I_atime[:], date)
	copy(Inode0.I_ctime[:], date)
	copy(Inode0.I_mtime[:], date)
	copy(Inode0.I_type[:], "0") //como es raiz es de tipo carpeta
	copy(Inode0.I_perm[:], "664")

	for i := int32(0); i < 15; i++ {
		Inode0.I_block[i] = -1
	}

	Inode0.I_block[0] = 0 //apunta al bloque 0

	//Crear el folder con la estructura
	// 	. 		| 0   -> actual (a si mismo)
	// 	..      | 0   -> el padre
	//users.txt | 1
	//			|-1

	var folderBlock0 Structs.Folderblock //Bloque0 -> carpetas
	folderBlock0.B_content[0].B_inodo = 0
	copy(folderBlock0.B_content[0].B_name[:], ".")
	folderBlock0.B_content[1].B_inodo = 0
	copy(folderBlock0.B_content[1].B_name[:], "..")
	folderBlock0.B_content[2].B_inodo = 1
	copy(folderBlock0.B_content[2].B_name[:], "users.txt")
	folderBlock0.B_content[3].B_inodo = -1

	//Inode1 que es el que contiene el archivo (Bloque 0 apunta a este nuevo inodo)
	var Inode1 Structs.Inode
	Inode1.I_uid = 1
	Inode1.I_gid = 1
	Inode1.I_size = int32(binary.Size(Structs.Folderblock{}))
	copy(Inode1.I_atime[:], date)
	copy(Inode1.I_ctime[:], date)
	copy(Inode1.I_mtime[:], date)
	copy(Inode1.I_type[:], "1") //es del archivo
	copy(Inode0.I_perm[:], "664")
	for i := int32(0); i < 15; i++ {
		Inode1.I_block[i] = -1
	}
	//Inode1 apunta al bloque1 (en este caso el bloque1 contiene el archivo)
	Inode1.I_block[0] = 1
	data := "1,G,root\n1,U,root,root,123\n"
	var fileBlock1 Structs.Fileblock //Bloque1 -> archivo
	copy(fileBlock1.B_content[:], []byte(data))

	//resumen
	//Inodo 0 -> Bloque 0 -> Inodo1 -> bloque1 (archivo)
	//Similar a las siguientes instrucciones analogas a la tarea3
	//Crear la carpeta raiz /
	//crear el archivo users.txt

	//fmt.Println("Superbloque: ", newSuperBloque)
	//fmt.Println("Journal: ", newJournal)

	// Escribir el superbloque
	Utilities.WriteObject(file, newSuperBloque, int64(particion.Start))

	//Escribir el Journaling
	Utilities.WriteObject(file, newJournal, int64(particion.Start+int32(binary.Size(Structs.Superblock{}))))

	//escribir el bitmap de inodos
	Utilities.WriteObject(file, byte(1), int64(newSuperBloque.S_bm_inode_start))
	Utilities.WriteObject(file, byte(1), int64(newSuperBloque.S_bm_inode_start+1)) //Se escribieron dos inode

	//escribir el bitmap de bloques (se usaron dos bloques)
	Utilities.WriteObject(file, byte(1), int64(newSuperBloque.S_bm_block_start))
	Utilities.WriteObject(file, byte(1), int64(newSuperBloque.S_bm_block_start+1))

	//escribir inodes
	//Inode0
	Utilities.WriteObject(file, Inode0, int64(newSuperBloque.S_inode_start))
	//Inode1
	Utilities.WriteObject(file, Inode1, int64(newSuperBloque.S_inode_start+int32(binary.Size(Structs.Inode{}))))

	//Escribir bloques
	//bloque0
	Utilities.WriteObject(file, folderBlock0, int64(newSuperBloque.S_block_start))
	//bloque1
	Utilities.WriteObject(file, fileBlock1, int64(newSuperBloque.S_block_start+int32(binary.Size(Structs.Fileblock{}))))
	// Fin crear EXT3
	fmt.Println("La particion con nombre " + string(particion.Name[:]) + " fue formateada con exito con EXT3")
	return "La particion con nombre " + Utilities.EliminartIlegibles(string(particion.Name[:])) + " fue formateada con exito con EXT3"
}
