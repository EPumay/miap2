package Analyzer

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"proyecto1/DiskManagement"
	"proyecto1/FileSystem"
	"proyecto1/Structs"
	"proyecto1/User"
	"proyecto1/Utilities"
	"strings"
)

func fn_mkdisk(params string) string {
	// Definir flag
	var respuesta string
	fs := flag.NewFlagSet("mkdisk", flag.ExitOnError)
	size := fs.Int("size", 0, "Tamaño") //nombre, valor por defecto, descripcion
	fit := fs.String("fit", "ff", "Ajuste")
	unit := fs.String("unit", "m", "Unidad")
	path := fs.String("path", "", "Ruta")

	fs.Parse(os.Args[1:]) //parsea los argumentos de la línea de comandos

	// Encontrar la flag en el input
	matches := re.FindAllStringSubmatch(params, -1) //encuentra todas las coincidencias de la expresión regular en el input

	// Process the input
	for _, match := range matches {
		flagName := strings.ToLower(match[1]) //guarda el nombre de la flag
		flagValue := match[2]                 //guarda el valor de la flag

		flagValue = strings.Trim(flagValue, "\"") //elimina las comillas del valor de la flag

		switch flagName {
		case "size", "fit", "unit", "path": //compara el nombre de la flag
			fs.Set(flagName, flagValue) //almacena el valor de la flag
		default:
			fmt.Println("Error: Parametro desconocido")
			return "\n Error: Parametro desconocido"
		}
	}
	//pasar flags a minisculas menos path
	*fit = strings.ToLower(*fit)
	*unit = strings.ToLower(*unit)

	if *size <= 0 {
		fmt.Println("Error: Size must be greater than 0")
		respuesta = "Error: Size must be greater than 0"
		return respuesta
	}

	if *fit != "bf" && *fit != "ff" && *fit != "wf" {
		fmt.Println("Error: Fit must be 'bf', 'ff', or 'wf'")
		respuesta = "Error: Fit must be 'bf', 'ff', or 'wf'"
		return respuesta
	}

	if *unit != "k" && *unit != "m" {
		fmt.Println("Error: Unit must be 'k' or 'm'")
		respuesta = "Error: Unit must be 'k' or 'm'"
		return respuesta
	}

	if *path == "" {
		fmt.Println("Error: Path is required")
		respuesta = "Error: Path is required"
		return respuesta
	}

	respuesta = DiskManagement.Mkdisk(*size, *fit, *unit, *path)
	return respuesta
}

func fn_rmdisk(input string) (respuesta string) {
	fs := flag.NewFlagSet("rmdisk", flag.ExitOnError)
	path := fs.String("path", "", "Ruta")
	fs.Parse(os.Args[1:])
	matches := re.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		flagName := strings.ToLower(match[1])
		flagValue := match[2]
		flagValue = strings.Trim(flagValue, "\"")
		switch flagName {
		case "path":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag not found")
		}
	}
	if *path == "" {
		fmt.Println("Error: Path is required")
		return
	}
	respuesta = DiskManagement.Rmdisk(*path)
	return respuesta
}

func fn_fdisk(input string) (respuesta string) {
	fs := flag.NewFlagSet("fdisk", flag.ExitOnError)
	size := fs.Int("size", 0, "Tamaño")
	path := fs.String("path", "", "Ruta")
	name := fs.String("name", "", "Nombre")
	unit := fs.String("unit", "k", "Unidad")
	type_ := fs.String("type", "p", "Tipo")
	fit := fs.String("fit", "wf", "Ajuste")
	delete_ := fs.String("delete", "", "Eliminar particion (Fast/Full)")

	// Parsear los flags
	fs.Parse(os.Args[1:])

	// Encontrar los flags en el input
	matches := re.FindAllStringSubmatch(input, -1)

	// Procesar el input
	for _, match := range matches {
		flagName := strings.ToLower(match[1]) // Convertir a minúsculas
		flagValue := match[2]                 // Obtener el valor de la flag

		flagValue = strings.Trim(flagValue, "\"")

		switch flagName {
		case "size", "fit", "unit", "path", "name", "type", "delete":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag not found")
		}
	}

	*name = strings.ToLower(*name)
	*unit = strings.ToLower(*unit)
	*type_ = strings.ToLower(*type_)
	*fit = strings.ToLower(*fit)
	*delete_ = strings.ToLower(*delete_)
	fmt.Print(*fit)

	if *delete_ != "" {
		if *path == "" || *name == "" {
			fmt.Println("Error: Path y Name son obligatorios para eliminar una particion")
			respuesta = "Error: Path y Name son obligatorios para eliminar una particion"
			return respuesta
		}
		respuesta = DiskManagement.DeletePartition(*path, *name, *delete_)
		return respuesta
	}

	// Convertir el nombre y la unidad a minúsculas

	// Validaciones
	if *size <= 0 {
		fmt.Println("Error: Size must be greater than 0")
		respuesta = "Error: Size must be greater than 0"
		return respuesta
	}

	if *path == "" {
		fmt.Println("Error: Path is required")
		return "Error: Path is required"
	}

	// Si no se proporcionó un fit, usar el valor predeterminado "w"
	if *fit == "" {
		*fit = "w"
	}

	if *fit != "bf" && *fit != "ff" && *fit != "wf" {
		fmt.Println("Error: Fit must be 'bf', 'ff', or 'wf'")
		respuesta = "Error: Fit must be 'bf', 'ff', or 'wf'"
		return respuesta
	}

	if *unit != "k" && *unit != "m" && *unit != "b" {
		fmt.Println("Error: Unit must be 'k', 'm', or 'b'")
		return "Error: Unit must be 'k' or 'm' or 'b'"
	}

	if *type_ != "p" && *type_ != "e" && *type_ != "l" {
		fmt.Println("Error: Type must be 'p', 'e', or 'l'")
		return "Error: Type must be 'p', 'e', or 'l'"
	}

	// Llamar a la función
	respuesta = DiskManagement.Fdisk(*size, *path, *name, *unit, *type_, *fit)
	return respuesta
}

func fn_mount(params string) (respuesta string) {
	fs := flag.NewFlagSet("mount", flag.ExitOnError)
	path := fs.String("path", "", "Ruta")
	name := fs.String("name", "", "Nombre de la partición")

	fs.Parse(os.Args[1:])
	matches := re.FindAllStringSubmatch(params, -1)

	for _, match := range matches {
		flagName := strings.ToLower(match[1]) // Convertir a minúsculas
		flagValue := match[2]                 // Obtener el valor de la flag
		flagValue = strings.Trim(flagValue, "\"")
		fs.Set(flagName, flagValue)
	}

	*name = strings.ToLower(*name)

	if *path == "" || *name == "" {
		fmt.Println("Error: Path y Name son obligatorios")
		return
	}

	respuesta = DiskManagement.Mount(*path, *name)
	return respuesta
}

func fn_rep(input string) (respuesta string) {
	fs := flag.NewFlagSet("rep", flag.ExitOnError)
	name := fs.String("name", "", "Nombre del reporte a generar (mbr, disk, inode, block, bm_inode, bm_block, sb, file, ls)")
	path := fs.String("path", "", "Ruta donde se generará el reporte")
	id := fs.String("id", "", "ID de la partición")
	pathFileLs := fs.String("path_file_ls", "", "Nombre del archivo o carpeta para reportes file o ls") // Parámetro opcional

	// Parsear los parámetros de entrada
	matches := re.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		flagName := strings.ToLower(match[1]) // Convertir a minúsculas
		flagValue := strings.Trim(match[2], "\"")

		switch flagName {
		case "name", "path", "id", "path_file_ls":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag no encontrada:", flagName)
		}
	}

	*name = strings.ToLower(*name)
	*id = strings.ToLower(*id)

	// Verificar los parámetros obligatorios
	if *name == "" || *path == "" || *id == "" {
		fmt.Println("Error: 'name', 'path' y 'id' son parámetros obligatorios.")
		return "Error: 'name', 'path' y 'id' son parámetros obligatorios."
	}

	// Verificar si el disco está montado usando DiskManagement
	mounted := false
	var diskPath string
	for _, partitions := range DiskManagement.GetMountedPartitions() {
		for _, partition := range partitions {
			if partition.Id == *id {
				mounted = true
				diskPath = partition.Path
				break
			}
		}
	}

	if !mounted {
		fmt.Println("Error: La partición con ID", *id, "no está montada.")
		return "Error: La partición con ID " + *id + " no está montada."
	}

	// Crear la carpeta si no existe
	reportsDir := filepath.Dir(*path)
	err := os.MkdirAll(reportsDir, os.ModePerm)
	if err != nil {
		fmt.Println("Error al crear la carpeta:", reportsDir)
		return "Error al crear la carpeta: " + reportsDir
	}

	// Generar el reporte según el tipo de reporte solicitado
	switch *name {
	case "mbr":
		// Abrir el archivo binario del disco montado
		file, err := Utilities.OpenFile(diskPath)
		if err != nil {
			fmt.Println("Error: No se pudo abrir el archivo en la ruta:", diskPath)
			return "Error: No se pudo abrir el archivo en la ruta: " + diskPath
		}
		defer file.Close()

		// Leer el objeto MBR desde el archivo binario
		var TempMBR Structs.MBR
		if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
			fmt.Println("Error: No se pudo leer el MBR desde el archivo")
			return "Error: No se pudo leer el MBR desde el archivo"
		}

		// Leer y procesar los EBRs si hay particiones extendidas
		var ebrs []Structs.EBR
		for i := 0; i < 4; i++ {
			if string(TempMBR.Partitions[i].Type[:]) == "e" { // Partición extendida
				fmt.Println("Partición extendida encontrada: ", string(TempMBR.Partitions[i].Name[:]))

				// El primer EBR está al inicio de la partición extendida
				ebrPosition := TempMBR.Partitions[i].Start
				ebrCounter := 1

				// Leer todos los EBRs dentro de la partición extendida
				for ebrPosition != -1 {
					fmt.Printf("Leyendo EBR en posición: %d\n", ebrPosition)
					var tempEBR Structs.EBR
					if err := Utilities.ReadObject(file, &tempEBR, int64(ebrPosition)); err != nil {
						fmt.Println("Error: No se pudo leer el EBR desde el archivo")
						break
					}

					// Añadir el EBR a la lista
					ebrs = append(ebrs, tempEBR)
					fmt.Printf("EBR %d leído. Start: %d, Size: %d, Next: %d, Name: %s\n", ebrCounter, tempEBR.Start, tempEBR.Size, tempEBR.Next, string(tempEBR.Name[:]))

					// Depuración: Mostrar el EBR leído
					Structs.PrintEBR(tempEBR)

					// Mover a la siguiente posición de EBR
					ebrPosition = tempEBR.Next
					ebrCounter++

					// Si no hay más EBRs, salir del bucle
					if ebrPosition == -1 {
						fmt.Println("No hay más EBRs en esta partición extendida.")
					}
				}
			}
		}

		// Generar el archivo .dot del MBR con EBRs
		reportPath := *path
		if err := Utilities.GenerateMBRReport(TempMBR, ebrs, reportPath, file); err != nil {
			fmt.Println("Error al generar el reporte MBR:", err)
		} else {
			fmt.Println("Reporte MBR generado exitosamente en:", reportPath)

			// Renderizar el archivo .dot a .jpg usando Graphviz
			dotFile := strings.TrimSuffix(reportPath, filepath.Ext(reportPath)) + ".dot"
			fmt.Println(dotFile)
			outputJpg := reportPath
			cmd := exec.Command("dot", "-Tjpg", dotFile, "-o", outputJpg)
			err = cmd.Run()
			if err != nil {
				fmt.Println("Error al renderizar el archivo .dot a imagen:", err)
			} else {
				fmt.Println("Imagen generada exitosamente en:", outputJpg)
			}
		}

	//CASE PARA EL REPORTE DISK
	case "disk":
		// Abrir el archivo binario del disco montado
		file, err := Utilities.OpenFile(diskPath)
		if err != nil {
			fmt.Println("Error: No se pudo abrir el archivo en la ruta:", diskPath)
			return "Error: No se pudo abrir el archivo en la ruta: " + diskPath
		}
		defer file.Close()

		// Leer el objeto MBR desde el archivo binario
		var TempMBR Structs.MBR
		if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
			fmt.Println("Error: No se pudo leer el MBR desde el archivo")
			return "Error: No se pudo leer el MBR desde el archivo"
		}

		// Leer y procesar los EBRs si hay particiones extendidas
		var ebrs []Structs.EBR
		for i := 0; i < 4; i++ {
			if string(TempMBR.Partitions[i].Type[:]) == "e" { // Partición extendida
				ebrPosition := TempMBR.Partitions[i].Start
				for ebrPosition != -1 {
					var tempEBR Structs.EBR
					if err := Utilities.ReadObject(file, &tempEBR, int64(ebrPosition)); err != nil {
						break
					}
					ebrs = append(ebrs, tempEBR)
					ebrPosition = tempEBR.Next
				}
			}
		}

		// Calcular el tamaño total del disco
		totalDiskSize := TempMBR.MbrSize

		// Generar el archivo .dot del DISK
		reportPath := *path
		if err := Utilities.GenerateDiskReport(TempMBR, ebrs, reportPath, file, totalDiskSize); err != nil {
			fmt.Println("Error al generar el reporte DISK:", err)
		} else {
			fmt.Println("Reporte DISK generado exitosamente en:", reportPath)

			// Renderizar el archivo .dot a .jpg usando Graphviz
			dotFile := strings.TrimSuffix(reportPath, filepath.Ext(reportPath)) + ".dot"
			outputJpg := reportPath
			cmd := exec.Command("dot", "-Tjpg", dotFile, "-o", outputJpg)
			err = cmd.Run()
			if err != nil {
				fmt.Println("Error al renderizar el archivo .dot a imagen:", err)
			} else {
				fmt.Println("Imagen generada exitosamente en:", outputJpg)
			}
		}

	case "inode":
		//llamada a la funcion para generar el reporte de inodo
		fmt.Println("Generando reporte de inodo")
	case "bm_inode":
		//llamada a la funcion para generar el reporte de bitmap de inodo
		fmt.Println("Generando reporte de bitmap de inodo")
		respuesta = FileSystem.BM_inode(*path, *id)

	case "bm_block":
		fmt.Println("Generando reporte de bitmap de bloque")
		respuesta = FileSystem.BM_Bloque(*path, *id)
	case "sb":
		//llamada a la funcion para generar el reporte de super bloque
		fmt.Println("Generando reporte de super bloque")
		respuesta = FileSystem.SuperBloque(*path, *id)
	case "block":
		//llamada a la funcion para generar el reporte de bloque
		fmt.Println("Generando reporte de bloque")
	case "file":
		//llamada a la funcion para generar el reporte de bloque
		fmt.Println("Generando reporte de bloque")
		respuesta = FileSystem.FILE(*path, *id, *pathFileLs)
	case "ls":
		//llamada a la funcion para generar el reporte de bloque
		fmt.Println("Generando reporte de bloque")
		respuesta = FileSystem.LS(*path, *id, *pathFileLs)

	default:
		fmt.Println("Error: Tipo de reporte no válido.")
	}
	respuesta = "Reporte generado exitosamente"

	return respuesta
}

func fn_logn(input string) (respuesta string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	user := fs.String("user", "", "Usuario")
	pass := fs.String("pass", "", "Contraseña")
	id := fs.String("id", "", "ID de la partición")

	fs.Parse(os.Args[1:])
	matches := re.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		flagName := strings.ToLower(match[1])  // Convertir a minúsculas
		flagValue := strings.ToLower(match[2]) // Obtener el valor de la flag
		flagValue = strings.Trim(flagValue, "\"")
		switch flagName {
		case "user", "pass", "id":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag no encontrada")
		}
	}

	//verificar que esten los parametros
	if *user == "" || *pass == "" || *id == "" {
		fmt.Println("Error: 'user', 'pass' y 'id' son parámetros obligatorios.")
		return "Error: 'user', 'pass' y 'id' son parámetros obligatorios."
	}
	respuesta = User.Login(*user, *pass, *id)
	return respuesta

}

func fn_mkfs(input string) (respuesta string) {
	fs := flag.NewFlagSet("mkfs", flag.ExitOnError)
	id := fs.String("id", "", "ID de la partición")
	type_ := fs.String("type", "full", "Tipo de formateo")
	fs.Parse(os.Args[1:])
	matches := re.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		flagName := strings.ToLower(match[1])  // Convertir a minúsculas
		flagValue := strings.ToLower(match[2]) // Obtener el valor de la flag
		flagValue = strings.Trim(flagValue, "\"")
		switch flagName {
		case "id":
			fs.Set(flagName, flagValue)
		case "type":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag no encontrada")
		}
	}
	if *id == "" {
		fmt.Println("Error: ID es obligatorio")
		return
	}
	fmt.Println("type", *type_)
	parametro := strings.Split(input, "-")
	respuesta = FileSystem.Mkfs(parametro)
	return respuesta
}
