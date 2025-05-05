package Structs

import (
	"fmt"
	"strings"
)

type MBR struct {
	MbrSize    int32        //tamaño del disco
	FechaC     [16]byte     //fecha de creacion
	Id         int32        //mbr_dsk_signature (random de forma unica)
	Fit        [1]byte      // B, F, W
	Partitions [4]Partition //mbr_partitions
}

func PrintMBR(data MBR) {
	fmt.Println("\n\t\tDisco")
	fmt.Printf("CreationDate: %s, fit: %s, size: %d, id: %d\n", string(data.FechaC[:]), string(data.Fit[:]), data.MbrSize, data.Id)
	for i := 0; i < 4; i++ {
		fmt.Printf("Partition %d: %s, %s, %d, %d, %s, %d\n", i, string(data.Partitions[i].Name[:]), string(data.Partitions[i].Type[:]), data.Partitions[i].Start, data.Partitions[i].Size, string(data.Partitions[i].Fit[:]), data.Partitions[i].Correlative)
	}
}

func GetIdMBR(m MBR) int32 {
	return m.Id
}

type Partition struct {
	Status      [1]byte //
	Type        [1]byte // P o E
	Fit         [1]byte // B, F o W
	Start       int32   // byte donde inicia la partición
	Size        int32   //
	Name        [16]byte
	Correlative int32 //desde -1
	Id          [4]byte
}

func (p *Partition) GetEnd() int32 {
	return p.Start + p.Size
}

type EBR struct {
	Status [1]byte //part_mount (si esta montada)
	Type   [1]byte
	Fit    byte     //part_fit
	Start  int32    //part_start
	Size   int32    //part_s
	Name   [16]byte //part_name
	Next   int32    //part_next
}

func PrintEBR(data EBR) {
	fmt.Println(fmt.Sprintf("Name: %s, fit: %c, start: %d, size: %d, next: %d, mount: %c",
		string(data.Name[:]),
		data.Fit,
		data.Start,
		data.Size,
		data.Next,
		data.Status))
}

var Montadas []mountAlready

type mountAlready struct {
	Id    string //Id de la particion
	PathM string //Path del disco al que pertenece la particion
}

// Ingresa la informacion al Struct
func AddMontadas(id string, path string) {
	Montadas = append(Montadas, mountAlready{Id: id, PathM: path})
}

func GetId(nombre string) string {
	//si existe id, no contiene bytes nulos
	posicionNulo := strings.IndexByte(nombre, 0)
	//si posicionNulo  no es -1, no existe id.
	if posicionNulo != -1 {
		nombre = "-"
	}
	return nombre
}

type Superblock struct {
	S_filesystem_type   int32    //numero que identifica el sistema de archivos usado //0->no formateada; 2->ext2; 3->ext3
	S_inodes_count      int32    //numero total de inods creados
	S_blocks_count      int32    //numero total de bloques creados
	S_free_blocks_count int32    //numero de bloques libres
	S_free_inodes_count int32    //numero de inodos libres
	S_mtime             [16]byte //ultima fecha en que el sistema fue montado "02/01/2006 15:04"
	S_umtime            [16]byte //ultima fecha en que el sistema fue desmontado "02/01/2006 15:04"
	S_mnt_count         int32    //numero de veces que se ha montado el sistema
	S_magic             int32    //valor que identifica el sistema de archivos (Sera 0xEF53)
	S_inode_size        int32    //tamaño de la etructura inodo
	S_block_size        int32    //tamaño de la estructura bloque
	S_first_ino         int32    //primer inodo libre
	S_first_blo         int32    //primer bloque libre
	S_bm_inode_start    int32    //inicio del bitmap de inodos
	S_bm_block_start    int32    //inicio del bitmap de bloques
	S_inode_start       int32    //inicio de la tabla de inodos
	S_block_start       int32    //inicio de la tabla de bloques
}

// INODO
type Inode struct {
	I_uid   int32     //ID del usuario propietario del archivo o carpeta
	I_gid   int32     //ID del grupo al que pertenece el archivo o carpeta
	I_size  int32     //tamaño del archivo en bytes
	I_atime [16]byte  //ultima fecha que se leyó el inodo sin modificarlo "02/01/2006 15:04"
	I_ctime [16]byte  //fecha en que se creo el inodo "02/01/2006 15:04"
	I_mtime [16]byte  //ultima fecha en la que se modifica el inodo "02/01/2006 15:04"
	I_block [15]int32 //-1 si no estan usados. los valores del arreglo son: primeros 12 -> bloques directo;: 13 -> bloque simple indirecto; 14->bloque doble indirecto; 15 -> bloque triple indirecto
	I_type  [1]byte   //1 -> archivo; 0 -> carpeta
	I_perm  [3]byte   //permisos del usuario o carpeta
}

type Folderblock struct {
	B_content [4]Content //contenido de la carpeta
}

type Content struct {
	B_name  [12]byte //nombre de carpeta/archivo
	B_inodo int32    //apuntador a un inodo asociado al archivo/carpeta
}

// BLOQUE DE ARCHIVOS
type Fileblock struct {
	B_content [64]byte //contenido del archivo
}

//USUARIOS

type UserInfo struct {
	IdPart   string //identificar la particion del usuario
	IdGrp    int32  //id del grupo al que pertenece el usuario
	IdUsr    int32  //id del usuario
	Nombre   string //saber que usuario es (identifica si es root o cualquir otro)
	Status   bool   //si esta iniciada la sesion
	DiskPath string //Path del disco
}

var UsuarioActual UserInfo

func SalirUsuario() {
	UsuarioActual.IdGrp = 0
	UsuarioActual.IdPart = ""
	UsuarioActual.IdUsr = 0
	UsuarioActual.Nombre = ""
	UsuarioActual.Status = false
	UsuarioActual.DiskPath = ""
}

//Para almacenar la informacion del usuario con sesion iniciada

// Valores por defecto al crear un objeto de esta estructura
// Id = ""
// Status = false -> false no hay sesion iniciada. true sesion iniciada
type Bite struct {
	Val [1]byte
}

func GetB_name(nombre string) string {
	posicionNulo := strings.IndexByte(nombre, 0)

	if posicionNulo != -1 {
		if posicionNulo != 0 {
			//tiene bytes nulos
			nombre = nombre[:posicionNulo]
		} else {
			//el  nombre esta vacio
			nombre = "-"
		}

	}
	return nombre //-1 el nombre no tiene bytes nulos
}

type Content_J struct {
	Operation [10]byte
	Path      [100]byte
	Content   [100]byte
	Date      [16]byte
}

type Journaling struct {
	Size      int32
	Ultimo    int32
	Contenido [50]Content_J
}

func GetName(nombre string) string {
	posicionNulo := strings.IndexByte(nombre, 0)
	//Si posicionNulo retorna -1 no hay bytes nulos
	if posicionNulo != -1 {
		//guarda la cadena hasta el primer byte nulo (elimina los bytes nulos)
		nombre = nombre[:posicionNulo]
	}
	return nombre
}
