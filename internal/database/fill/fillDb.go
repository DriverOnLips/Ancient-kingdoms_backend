package main

import (
	"fmt"
	"kingdoms/internal/database/connect"
	"kingdoms/internal/database/schema"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/icrowley/fake"
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

	fake.Seed(time.Now().UnixNano())
	fake.SetLang("ru")

	b, err := os.ReadFile("./files/defaultAvatarBase64.txt")
	if err != nil {
		fmt.Print(err)
	}

	defaultAvatar := string(b)

	var kingdoms []schema.Kingdom
	kingdomNames := make(map[string]bool)

	for i := 0; i < 200; i++ {
		kingdomName, kingdomCapital := getKingdomNameAndCapital()

		if _, exists := kingdomNames[kingdomName]; exists {
			continue
		}

		kingdomNames[kingdomName] = true
		kingdomArea := rand.Intn(100000)

		kingdom := schema.Kingdom{
			Name:        kingdomName,
			Area:        kingdomArea,
			Capital:     kingdomCapital,
			Image:       defaultAvatar,
			Description: getKingdomDescription(kingdomName, kingdomCapital, strconv.Itoa(kingdomArea)),
			State:       getKingdomState(),
		}
		kingdoms = append(kingdoms, kingdom)
	}

	result := db.CreateInBatches(&kingdoms, 1000)
	if result.Error != nil {
		log.Fatalf("Failed to bulk insert kingdoms: %v", result.Error)
	}
}
