package role

type Role int

const (
	Unknown Role = iota
	Buyer
	Manager
	Admin
)
