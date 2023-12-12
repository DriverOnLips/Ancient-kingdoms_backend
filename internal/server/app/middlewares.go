package app

import (
	"errors"
	role "kingdoms/internal/server/app/userRole"
	"kingdoms/internal/server/models/serverModels"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt"
)

const jwtPrefix = "Bearer "

func (a *Application) WithAuthCheck(assignedRoles ...role.Role) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		jwtStr := ctx.GetHeader("Authorization")

		if jwtStr == "" {
			var cookieErr error
			jwtStr, cookieErr = ctx.Cookie("kingdoms-token")
			if cookieErr != nil {
				ctx.AbortWithStatus(http.StatusBadRequest)
			}
		}

		if !strings.HasPrefix(jwtStr, jwtPrefix) { // если нет префикса то нас дурят!
			ctx.AbortWithStatus(http.StatusForbidden) // отдаем что нет доступа

			return // завершаем обработку
		}

		// отрезаем префикс
		jwtStr = jwtStr[len(jwtPrefix):]
		// проверяем jwt в блеклист редиса
		err := a.redis.CheckJWTInBlacklist(ctx.Request.Context(), jwtStr)
		if err == nil { // значит что токен в блеклисте
			ctx.AbortWithStatus(http.StatusForbidden)

			return
		}
		if !errors.Is(err, redis.Nil) { // значит что это не ошибка отсуствия - внутренняя ошибка
			ctx.AbortWithError(http.StatusInternalServerError, err)

			return
		}

		token, err := jwt.ParseWithClaims(jwtStr, &serverModels.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(a.config.JWT.Token), nil
		})
		if err != nil {
			ctx.AbortWithStatus(http.StatusForbidden)
			log.Println(err)

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
			log.Printf("role %d is not assigned in %d", myClaims.Role, assignedRoles)
			return
		}

		ctx.Set("role", myClaims.Role)
		ctx.Set("userUUID", myClaims.UserUUID)
	}

}
