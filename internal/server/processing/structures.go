package processing

import (
	"kingdoms/internal/database/schema"

	"gorm.io/datatypes"
)

type StructApplicationWithKingdoms struct {
	Application schema.RulerApplication
	Kingdoms    []KingdomFromApplication
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

type KingdomAddToApplication struct {
	ApplicationId uint
	KingdomId     uint
	From          datatypes.Date
	To            datatypes.Date
}

type DeleteKingdomFromApplication struct {
	ApplicationId uint
	KingdomId     uint
}
