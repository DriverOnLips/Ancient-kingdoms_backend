package serverModels

import (
	role "kingdoms/internal/server/app/userRole"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type JWTClaims struct {
	jwt.StandardClaims
	UserUUID uuid.UUID `json:"user_uuid"`
	Role     role.Role
	UserName string `json:"user_name"`
}
