package Analyzer

import (
	"fmt"
	"proyecto1/DiskManagement"
	"proyecto1/FileSystem"
	"proyecto1/User"
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`-(\w+)=("[^"]+"|\S+)`)

func getCommandAndParams(input string) (string, string) {
	parts := strings.Fields(input)
	if len(parts) > 0 {
		command := strings.ToLower(parts[0])
		params := strings.Join(parts[1:], " ")
		return command, params
	}
	return "", input

	/*Después de procesar la entrada:
	command será "mkdisk".
	params será "-size=3000 -unit=K -fit=BF -path=/home/bang/Disks/disk1.bin".*/
}

func Analyze(input string) string {
	command, params := getCommandAndParams(input)

	fmt.Println("Comando: ", command, " - ", "Parametro: ", params)

	respuesta := AnalyzeCommnad(command, params)

	return respuesta

}

func AnalyzeCommnad(command string, params string) string {
	var respuesta string
	if strings.Contains(command, "mkdisk") {
		fmt.Print("Comando: mkdisk\n")
		respuesta = fn_mkdisk(params)
	} else if strings.Contains(command, "rmdisk") {
		fmt.Print("Comando: rmdisk\n")
		respuesta = fn_rmdisk(params)
	} else if strings.Contains(command, "fdisk") {
		fmt.Print("Comando: fdisk\n")
		respuesta = fn_fdisk(params)
	} else if strings.Contains(command, "mounted") {
		fmt.Print("Comando: mounted\n")
		respuesta = DiskManagement.Mounted()
		print(respuesta)
	} else if strings.Contains(command, "mount") {
		fmt.Print("Comando: mount\n")
		respuesta = fn_mount(params)
	} else if strings.Contains(command, "rep") {
		respuesta = fn_rep(params)
	} else if strings.Contains(command, "login") {
		fmt.Print("Comando: login\n")
		respuesta = fn_logn(params)
	} else if strings.Contains(command, "mkfs") {
		fmt.Print("Comando: mkfs\n")
		respuesta = fn_mkfs(params)
	} else if strings.Contains(command, "logout") {
		fmt.Print("Comando: logout\n")
		respuesta = User.Logout()
	} else if strings.Contains(command, "mkdir") {
		fmt.Print("Comando: mkdir\n")
		parametros := strings.Split(params, "-")
		respuesta = FileSystem.Mkdir(parametros)
		fmt.Println(parametros)
	} else if strings.Contains(command, "unmount") {
		fmt.Print("Comando: mkdir\n")
		parametros := strings.Split(params, "-")
		respuesta = DiskManagement.Unmount(parametros)
		fmt.Println(parametros)
	} else if strings.Contains(command, "mkgrp") {
		parametros := strings.Split(params, "-")
		respuesta = User.Mkgrp(parametros)
	} else if strings.Contains(command, "mkusr") {
		parametros := strings.Split(params, "-")
		respuesta = User.Mkusr(parametros)
	} else if strings.Contains(command, "mkfile") {
		fmt.Print("Comando: mkfile\n")
		parametros := strings.Split(params, "-")
		respuesta = FileSystem.Mkfile(parametros)
	} else if strings.Contains(command, "cat") {
		fmt.Print("Comando: cat\n")
		parametros := strings.Split(params, "-")
		respuesta = FileSystem.Cat(parametros)
	}

	return respuesta
}
