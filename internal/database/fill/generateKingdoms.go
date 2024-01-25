package main

import (
	"math/rand"
	"strings"

	"github.com/icrowley/fake"
)

func getKingdomsType() string {
	words := []string{"государство", "княжество", "царство", "королевство", "владение",
		"объединение"}
	wordIndex := rand.Intn(len(words))

	return words[wordIndex]
}

func getKingdomCapital() string {
	return strings.TrimSpace(fake.City())
}

func getKingdomPrefix() string {
	words := []string{"Первое", "Второе", "Третье", "Четвертое", "Пятое",
		"Шестое", "Седьмое", "Восьмое", "Девятое", "Десятое", "Одиннадцатое",
		"Великое", "Всемогущее", "Прекрасное", "Богатое", "Островное",
		"Аграрное", "Пустынное", "Многонациональное", "Городское", "Деревенское"}
	wordIndex := rand.Intn(len(words))

	return words[wordIndex]
}

func getKingdomNameAndCapital() (string, string) {
	capital := getKingdomCapital()

	return getKingdomPrefix() + " " + capital + "ское " + getKingdomsType(), capital
}

func getKingdomDescription(name string, capital string, area string) string {
	return name + " имеет площадь " + area + ", ее столица - " + capital +
		". Да и в целом это классное княжество)"
}

func getKingdomState() string {
	index := rand.Intn(10)
	if index < 9 {
		return "Данные подтверждены"
	}

	return "Данные утеряны"
}
