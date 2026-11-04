package main

import (
	"fmt"
	"gatra-budget-backend/config"
	"gatra-budget-backend/seeds"
	"log"
	"os"

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

	if len(args) > 0 && args[0] == "demo-reset" {
		assumeYes := len(args) > 1 && args[1] == "--yes"
		if err := seeds.DemoReset(config.DB, os.Stdin, os.Stdout, assumeYes); err != nil {
			log.Fatalf("Error resetting demo data: %v", err)
		}
		return
	}

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
		{"operational_cost_types", func() error { return seeds.OperationalCostTypesSeeder(config.DB) }},
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
	case "operational_cost_types":
		err = seeds.OperationalCostTypesSeeder(config.DB)
	case "demo":
		err = seeds.DemoSeeder(config.DB)
	default:
		log.Fatalf("Unknown seeder: %s\n\nAvailable seeders:\n  - address_reference\n  - users\n  - operational_cost_types\n  - demo (sample data 2025-2026, not part of 'all')\n  - demo-reset (interactive, removes demo data)\n", name)
	}

	if err != nil {
		log.Fatalf("Error running seeder: %v", err)
	}
}
