package app

import (
	"errors"
	role "kingdoms/internal/server/app/userRole"
	"kingdoms/internal/server/models/responseModels"
	"kingdoms/internal/server/models/serverModels"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt"
)

const jwtPrefix = "Bearer"

func (a *Application) WithAuthCheck(assignedRoles ...role.Role) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		jwtStr, cookieErr := ctx.Cookie("kingdoms-token")
		if cookieErr != nil {
			response := responseModels.ResponseDefault{
				Code:    500,
				Status:  "error",
				Message: "error getting cookie",
				Body:    nil,
			}

			ctx.JSON(http.StatusInternalServerError, response)
			return
		}

		if !strings.HasPrefix(jwtStr, jwtPrefix) {
			response := responseModels.ResponseDefault{
				Code:    500,
				Status:  "error",
				Message: "error parsing jwt token: no prefix",
				Body:    nil,
			}

			ctx.JSON(http.StatusInternalServerError, response)
			return
		}

		jwtStr = jwtStr[len(jwtPrefix):]

		err := a.redis.CheckJWTInBlacklist(ctx.Request.Context(), jwtStr)
		if err == nil {
			response := responseModels.ResponseDefault{
				Code:    403,
				Status:  "error",
				Message: "not authorized: token in black list",
				Body:    nil,
			}

			ctx.JSON(http.StatusForbidden, response)
			return
		}
		if !errors.Is(err, redis.Nil) {
			response := responseModels.ResponseDefault{
				Code:    500,
				Status:  "error",
				Message: "server error: " + err.Error(),
				Body:    nil,
			}

			ctx.JSON(http.StatusInternalServerError, response)
			return
		}

		token, err := jwt.ParseWithClaims(jwtStr, &serverModels.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(a.config.JWT.Token), nil
		})
		if err != nil {
			response := responseModels.ResponseDefault{
				Code:    403,
				Status:  "error",
				Message: "error parsing jwt token: error parsing with claims:" + err.Error(),
				Body:    nil,
			}

			ctx.JSON(http.StatusForbidden, response)
			return
		}

		myClaims := token.Claims.(*serverModels.JWTClaims)

		isAssigned := false

		for _, oneOfAssignedRole := range assignedRoles {
			if myClaims.Role == oneOfAssignedRole {
				isAssigned = true
				break
			}
		}

		if !isAssigned {
			ctx.AbortWithStatus(http.StatusForbidden)
			response := responseModels.ResponseDefault{
				Code:    403,
				Status:  "error",
				Message: "role " + string(rune(myClaims.Role)) + " is not assigned",
				Body:    nil,
			}

			ctx.JSON(http.StatusForbidden, response)
			return
		}

		// ctx.Set("user_role", myClaims.Role)
		// ctx.Set("userUUID", myClaims.UserUUID)
	}
}
