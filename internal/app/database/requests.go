package database

import "gorm.io/datatypes"

type GetKingdomsRequest struct {
	Ruler string
	State string
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
	Ruler          Ruler
	Kingdom        Kingdom
	BeginGoverning datatypes.Date
}
