package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	passwords := []string{"admin123", "manager123"}
	for _, p := range passwords {
		hash, _ := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
		fmt.Printf("Password: %s\nHash: %s\n\n", p, hash)
	}
}
