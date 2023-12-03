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

		for _, oneOfAssignedRole := range assignedRoles {
			if myClaims.Role == oneOfAssignedRole {
				ctx.Next()
			}
		}
		ctx.AbortWithStatus(http.StatusForbidden)
		log.Printf("role %s is not assigned in %s", myClaims.Role, assignedRoles)

		return

	}

}
