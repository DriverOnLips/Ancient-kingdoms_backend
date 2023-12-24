package processing

import (
	"kingdoms/internal/database/schema"

	"gorm.io/datatypes"
)

type StructApplicationWithKingdoms struct {
	ApplicationId string
	Kingdoms      []KingdomFromApplication
}

type KingdomFromApplication struct {
	Kingdom schema.Kingdom
	From    datatypes.Date
	To      datatypes.Date
}

type AsyncStructApplication struct {
	Id    uint `json:"Id"`
	Check bool `json:"Check"`
}

type ApplicationToUpdate struct {
	Id    uint
	State string
}
