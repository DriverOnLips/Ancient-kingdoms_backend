package schema

import (
	role "kingdoms/internal/server/app/userRole"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Kingdom struct {
	Id          uint   `gorm:"primaryKey;AUTO_INCREMENT"`
	Name        string `gorm:"type:varchar(100);unique;not null"`
	Area        int    `gorm:"not null"`
	Capital     string `gorm:"type:varchar(50);not null"`
	Image       string `gorm:"type:bytea"`
	Description string `gorm:"size:255"`
	State       string `gorm:"type:varchar(50);not null"`
}

type User struct {
	Id       uint      `gorm:"primaryKey;AUTO_INCREMENT"`
	UUID     uuid.UUID `gorm:"type:uuid"`
	Name     string    `json:"name"`
	Role     role.Role `sql:"type:string"`
	Password string
}

type RulerApplication struct {
	Id           uint           `gorm:"primaryKey;AUTO_INCREMENT"`
	State        string         `gorm:"type:varchar(50);not null"`
	DateCreate   datatypes.Date `gorm:"not null"`
	DateSend     datatypes.Date
	DateComplete datatypes.Date
	Ruler        string `gorm:"type:varchar(50);not null"`
	CreatorRefer int    `gorm:"not null"`
	Creator      User   `gorm:"foreignKey:CreatorRefer"`
	// ModeratorRefer int    `gorm:"not null"`
	// Moderator      User   `gorm:"foreignKey:ModeratorRefer"`
	Check bool `gorm:"type:boolean"`
}

type Kingdom2Application struct {
	Id               uint             `gorm:"primaryKey;AUTO_INCREMENT"`
	KingdomRefer     int              `gorm:"not null"`
	Kingdom          Kingdom          `gorm:"foreignKey:KingdomRefer"`
	ApplicationRefer int              `gorm:"not null"`
	Application      RulerApplication `gorm:"foreignKey:ApplicationRefer"`
	From             datatypes.Date   `gorm:"not null"`
	To               datatypes.Date   `gorm:"not null"`
}
