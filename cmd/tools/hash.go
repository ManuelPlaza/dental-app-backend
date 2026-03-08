package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Uso: go run cmd/tools/hash.go <contraseña>")
		os.Exit(1)
	}
	password := os.Args[1]
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error generando hash:", err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}
