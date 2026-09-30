package main

import (
	"fmt"
	"log"
	"os"
	"gatra-budget-backend/config"
	"gatra-budget-backend/seeds"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	fmt.Println("Connecting to database...")
	config.ConnectDB()
	fmt.Println("Database connected")
	fmt.Println("")

	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("Running all seeders...")
		fmt.Println("==========================================")
		runAllSeeders()
	} else {
		seederName := args[0]
		fmt.Printf("Running seeder: %s\n", seederName)
		fmt.Println("==========================================")
		runSpecificSeeder(seederName)
	}

	fmt.Println("")
	fmt.Println("==========================================")
	fmt.Println("Seeding completed successfully!")
}

func runAllSeeders() {
	seeders := []struct {
		name string
		fn   func() error
	}{
		{"address_reference", func() error { return seeds.AddressReferenceSeeder(config.DB) }},
		{"users", func() error { return seeds.UsersSeeder(config.DB) }},
	}

	for _, seeder := range seeders {
		fmt.Printf("\nRunning %s seeder...\n", seeder.name)
		if err := seeder.fn(); err != nil {
			log.Fatalf("Error running %s seeder: %v", seeder.name, err)
		}
	}
}

func runSpecificSeeder(name string) {
	var err error

	switch name {
	case "address_reference":
		err = seeds.AddressReferenceSeeder(config.DB)
	case "users":
		err = seeds.UsersSeeder(config.DB)
	default:
		log.Fatalf("Unknown seeder: %s\n\nAvailable seeders:\n  - address_reference\n  - users\n", name)
	}

	if err != nil {
		log.Fatalf("Error running seeder: %v", err)
	}
}
