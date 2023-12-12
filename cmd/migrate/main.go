package main

import (
	"fmt"
	"kingdoms/internal/database/connect"
	"kingdoms/internal/database/schema"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(connect.FromEnv()), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect database! Error:", err)
		return
	}

	// Migrate the schema
	err = MigrateSchema(db)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func MigrateSchema(db *gorm.DB) error {
	err := db.AutoMigrate(&schema.Kingdom{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&schema.User{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&schema.Campaign{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&schema.Kingdom4campaign{})
	if err != nil {
		return err
	}

	return nil
}

// func MigrateKingdom(db *gorm.DB) error {
// 	err := db.AutoMigrate(&schema.Kingdom{})
// 	if err != nil {
// 		fmt.Println("Error migrating Kingdom to db")
// 		return err
// 	}

// 	return nil
// }

// func MigrateUser(db *gorm.DB) error {
// 	err := db.AutoMigrate(&schema.User{})
// 	if err != nil {
// 		fmt.Println("Error migrating User to db")
// 		return err
// 	}

// 	return nil
// }

// func MigrateCampaign(db *gorm.DB) error {
// 	err := db.AutoMigrate(&schema.Campaign{})
// 	if err != nil {
// 		fmt.Println("Error migrating Campaign to db")
// 		return err
// 	}

// 	return nil
// }

// func MigrateKingdom4campaign(db *gorm.DB) error {
// 	err := db.AutoMigrate(&schema.Kingdom4campaign{})
// 	if err != nil {
// 		fmt.Println("Error migrating Kingdom4campaign to db")
// 		return err
// 	}

// 	return nil
// }
