package requests

import (
	"kingdoms/internal/database/schema"

	"gorm.io/datatypes"
)

type GetKingdomsRequest struct {
	KingdomName string
	RulerName   string
	State       string
}

type GetRulersRequest struct {
	Num   int
	State string
}

type RulerStateChangeRequest struct {
	ID    int
	State string
	User  string
}

type CreateRulerForKingdomRequest struct {
	Ruler          schema.Ruler
	Kingdom        schema.Kingdom
	BeginGoverning datatypes.Date
}
