package schema

import (
	role "kingdoms/internal/server/app/userRole"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Kingdom struct {
	ID          uint   `gorm:"primaryKey;AUTO_INCREMENT"`
	Name        string `gorm:"type:varchar(50);unique;not null"`
	Area        int    `gorm:"not null"`
	Capital     string `gorm:"type:varchar(50);not null"`
	Image       string `gorm:"type:bytea"`
	Description string `gorm:"size:255"`
	State       string `gorm:"type:varchar(50);not null"`
}

type User struct {
	ID       uint      `gorm:"primaryKey;AUTO_INCREMENT"`
	UUID     uuid.UUID `gorm:"type:uuid"`
	Name     string    `json:"name"`
	Role     role.Role `sql:"type:string"`
	Password string
}

type Campaign struct {
	ID          uint           `gorm:"primaryKey;AUTO_INCREMENT"`
	UserRefer   int            `gorm:"not null"`
	User        User           `gorm:"foreignKey:UserRefer"`
	KingName    string         `gorm:"type:varchar(50);unique;not null"`
	State       string         `gorm:"type:varchar(50);not null"`
	Development datatypes.Date `gorm:"not null"`
	Begin       datatypes.Date
	End         datatypes.Date
}

type Kingdom4campaign struct {
	ID            uint     `gorm:"primaryKey;AUTO_INCREMENT"`
	CampaignRefer int      `gorm:"not null"`
	Campaign      Campaign `gorm:"foreignKey:CampaignRefer"`
	KingdomRefer  int      `gorm:"not null"`
	Kingdom       Kingdom  `gorm:"foreignKey:KingdomRefer"`
	NumKingdoms   int      `gorm:"not null"`
}
