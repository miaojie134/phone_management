package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := []byte("admin") // 你要设置的密码
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	fmt.Printf("Username: admin\n")
	fmt.Printf("Hashed Password: %s\n", string(hashedPassword))
}

// INSERT INTO users (username, password_hash, role, created_at, updated_at)
// VALUES ('admin', '$2a$10$/lpVGyBdxr9Px8aifH7K/.ozClF0Di54vuV0.tDllRQouMk.jj.dG', 'admin', strftime('%Y-%m-%d %H:%M:%S', 'now'), strftime('%Y-%m-%d %H:%M:%S', 'now'));
