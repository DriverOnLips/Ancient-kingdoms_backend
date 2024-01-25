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

	err = db.AutoMigrate(&schema.RulerApplication{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&schema.Kingdom2Application{})
	if err != nil {
		return err
	}

	return nil
}
