package app

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	config "kingdoms/internal/config"
	"kingdoms/internal/database/connect"
	"kingdoms/internal/database/schema"
	role "kingdoms/internal/server/app/userRole"
	"kingdoms/internal/server/models/responseModels"
	"kingdoms/internal/server/models/serverModels"
	"kingdoms/internal/server/redis"

	"kingdoms/internal/server/processing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

const ASYNC_KEY = "secret"

type Application struct {
	config *config.Config
	repo   *processing.Repository
	redis  *redis.Client
	r      *gin.Engine
}

func New(ctx context.Context) (*Application, error) {
	cfg, err := config.NewConfig(ctx)
	if err != nil {
		return nil, err
	}

	repo, err := processing.New(connect.FromEnv())
	if err != nil {
		return nil, err
	}

	redisClient, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		return nil, err
	}

	return &Application{
		config: cfg,
		repo:   repo,
		redis:  redisClient,
	}, nil
}

func (a *Application) Run() error {
	log.Println("application start running")
	a.StartServer()
	log.Println("application shut down")

	return nil
}

func (a *Application) StartServer() {
	log.Println("Server started")

	a.r = gin.Default()

	a.r.GET("kingdoms", a.getKingdomsFeed)
	a.r.GET("kingdom", a.getKingdom)
	a.r.GET("applications", a.getApplications)
	a.r.GET("application/with_kingdoms", a.getApplicationWithKingdoms)

	a.r.GET("async/application", a.asyncGetApplication)
	a.r.PUT("async/application", a.asyncPutApplicationInfo)

	// a.r.POST("kingdom/add", a.addKingdom)
	// a.r.PUT("kingdom/edit", a.editKingdom)
	// a.r.PUT("kingdom/ruler_to_kingdom", a.CreateRulerForKingdom)

	// // a.r.Use(a.WithAuthCheck(role.Moderator, role.Admin)).PUT("ruler/edit", a.editRuler)
	// a.r.PUT("ruler/edit", a.editRuler)
	// a.r.PUT("ruler/state_change/moderator", a.rulerStateChangeModerator)
	// a.r.PUT("ruler/state_change/user", a.rulerStateChangeUser)

	// a.r.PUT("kingdom/delete/:kingdom_name", a.deleteKingdom)
	// a.r.PUT("kingdom/ruler/:ruler_name", a.deleteRuler)

	// a.r.DELETE("kingdom_ruler_delete/:kingdom_name/:ruler_name/:ruling_id", a.deleteKingdomRuler)

	a.r.GET("login", a.checkLogin)
	a.r.POST("login", a.login)
	a.r.POST("signup", a.signup)
	a.r.DELETE("logout", a.logout)

	a.r.Run(":8000")

	log.Println("Server is down")
}

func (a *Application) asyncGetApplication(ctx *gin.Context) {
	key := ctx.GetHeader("AsyncKey")
	if key != ASYNC_KEY {
		response := responseModels.ResponseDefault{
			Code:    403,
			Status:  "error",
			Message: "error getting async server key",
			Body:    nil,
		}

		ctx.JSON(http.StatusForbidden, response)
		return
	}

	applicationId := ctx.Query("id")
	if applicationId == "" {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error no id provided",
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	applications, err := a.repo.AsyncGetApplication(applicationId)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting necessary applications: " + err.Error(),
			Body:    nil,
		}
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdoms from application found",
		Body:    applications,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) asyncPutApplicationInfo(ctx *gin.Context) {
	key := ctx.GetHeader("AsyncKey")
	if key != ASYNC_KEY {
		response := responseModels.ResponseDefault{
			Code:    403,
			Status:  "error",
			Message: "error getting async server key",
			Body:    nil,
		}

		ctx.JSON(http.StatusForbidden, response)
		return
	}

	var applicationToPut processing.AsyncStructApplication
	if err := ctx.BindJSON(&applicationToPut); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing application:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err := a.repo.AsyncPutApplicationInfo(applicationToPut)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: err.Error(),
			Body:    nil,
		}
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "appliction updated successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) getKingdomsFeed(ctx *gin.Context) {
	kingdomName := ctx.Query("kingdom_name") // TODO окно для пагинации

	kingdoms, err := a.repo.GetKingdoms(kingdomName)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting necessary kingdoms: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)

		return
	}

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdoms found",
		Body:    kingdoms,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) getKingdom(ctx *gin.Context) {
	kingdomID, err := strconv.Atoi(ctx.Query("id"))
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting kingdom by ID: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)

		return
	}

	var kingdom schema.Kingdom
	kingdom.Id = uint(kingdomID)

	kingdom, err = a.repo.GetKingdom(kingdom)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting necessary kingdom: " + err.Error(),
			Body:    nil,
		}
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom found",
		Body:    kingdom,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) getApplications(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.UserName)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting user by name: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	applicationId := ctx.Query("id")

	applications, err := a.repo.GetApplications(*user, applicationId)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting necessary applications: " + err.Error(),
			Body:    nil,
		}
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "applications found",
		Body:    applications,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) getApplicationWithKingdoms(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.UserName)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting user by name: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	applicationId := ctx.Query("id")
	if applicationId == "" {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error no id provided",
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	applications, err := a.repo.GetApplicationWithKingdoms(*user, applicationId)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting necessary kingdoms form application: " + err.Error(),
			Body:    nil,
		}
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdoms from application found",
		Body:    applications,
	}

	ctx.JSON(http.StatusOK, response)
}

// func (a *Application) getRuler(ctx *gin.Context) {
// 	var ruler schema.Ruler
// 	if err := ctx.BindJSON(&ruler); err != nil {
// 		ctx.String(http.StatusBadRequest, "error parsing ruler:"+err.Error())
// 		return
// 	}

// 	necessaryRuler, err := a.repo.GetRuler(ruler)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error getting ruler:"+err.Error())
// 		return
// 	}
// 	if necessaryRuler == (schema.Ruler{}) {
// 		ctx.String(http.StatusNotFound, "no necessary ruler")
// 		return
// 	}

// 	ctx.JSON(http.StatusFound, necessaryRuler)
// }

// func (a *Application) addKingdom(ctx *gin.Context) {
// 	var kingdom schema.Kingdom
// 	if err := ctx.BindJSON(&kingdom); err != nil {
// 		ctx.String(http.StatusBadRequest, "error parsing kingdom:"+err.Error())
// 		return
// 	}

// 	err := a.repo.CreateKingdom(kingdom)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error creating kingdom:"+err.Error())
// 		return
// 	}

// 	ctx.String(http.StatusCreated, "creating kingdom done successfully")
// }

// func (a *Application) editKingdom(ctx *gin.Context) {
// 	var kingdom schema.Kingdom
// 	if err := ctx.BindJSON(&kingdom); err != nil {
// 		ctx.String(http.StatusBadRequest, "error parsing ruler")
// 		return
// 	}

// 	err := a.repo.EditKingdom(kingdom)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error editing kingdom:"+err.Error())
// 		return
// 	}

// 	ctx.JSON(http.StatusNoContent, kingdom)
// }

// func (a *Application) CreateRulerForKingdom(ctx *gin.Context) {
// 	var requestBody requestsModels.CreateRulerForKingdomRequest
// 	if err := ctx.BindJSON(&requestBody); err != nil {
// 		ctx.String(http.StatusBadRequest, "error parsing kingdom:"+err.Error())
// 		return
// 	}

// 	err := a.repo.CreateRulerForKingdom(requestBody)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error ruler for kingdom additing:"+err.Error())
// 		return
// 	}

// 	ctx.String(http.StatusNoContent, "additing done successfully")
// }

// func (a *Application) editRuler(ctx *gin.Context) {
// 	var ruler schema.Ruler
// 	if err := ctx.BindJSON(&ruler); err != nil {
// 		ctx.String(http.StatusBadRequest, "error parsing ruler:"+err.Error())
// 		return
// 	}

// 	err := a.repo.EditRuler(ruler)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error editing ruler:"+err.Error())
// 		return
// 	}

// 	ctx.String(http.StatusNoContent, "edditing ruler done successfully")
// }

// func (a *Application) rulerStateChangeModerator(ctx *gin.Context) {
// 	var requestBody requestsModels.RulerStateChangeRequest
// 	if err := ctx.BindJSON(&requestBody); err != nil {
// 		ctx.String(http.StatusBadRequest, "error parsing request body:"+err.Error())
// 		return
// 	}

// 	userRole, err := a.repo.GetUserRole(requestBody.User)
// 	if err != nil {
// 		ctx.String(http.StatusBadRequest, "error getting user role:"+err.Error())
// 		return
// 	}
// 	if userRole != role.Admin {
// 		ctx.String(http.StatusUnauthorized, "no enouth rules for executing this operation")
// 		return
// 	}

// 	err = a.repo.RulerStateChange(requestBody.ID, requestBody.State)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error ruler state changing:"+err.Error())
// 		return
// 	}

// 	ctx.String(http.StatusNoContent, "ruler state changing done successfully")
// }

// func (a *Application) rulerStateChangeUser(ctx *gin.Context) {
// 	var requestBody requestsModels.RulerStateChangeRequest
// 	if err := ctx.BindJSON(&requestBody); err != nil {
// 		ctx.String(http.StatusBadRequest, "error parsing request body:"+err.Error())
// 		return
// 	}

// 	userRole, err := a.repo.GetUserRole(requestBody.User)
// 	if err != nil {
// 		ctx.String(http.StatusBadRequest, "error getting user role:"+err.Error())
// 		return
// 	}
// 	if userRole != role.Admin && userRole != role.Manager {
// 		ctx.String(http.StatusUnauthorized, "no enouth rules for executing this operation")
// 		return
// 	}

// 	err = a.repo.RulerStateChange(requestBody.ID, requestBody.State)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error ruler state changing:"+err.Error())
// 		return
// 	}

// 	ctx.String(http.StatusNoContent, "ruler state changing done successfully")
// }

// func (a *Application) deleteKingdom(ctx *gin.Context) {
// 	kingdomName := ctx.Param("kingdom_name")

// 	err := a.repo.DeleteKingdom(kingdomName)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error deleting kingdom:"+err.Error())
// 		return
// 	}

// 	ctx.String(http.StatusNoContent, "deleting kingdom done successfully")
// }

// func (a *Application) deleteRuler(ctx *gin.Context) {
// 	rulerName := ctx.Param("ruler_name")

// 	err := a.repo.DeleteRuler(rulerName)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error deleting ruler:"+err.Error())
// 		return
// 	}

// 	ctx.String(http.StatusNoContent, "deleting ruler done successfully")
// }

// func (a *Application) deleteKingdomRuler(ctx *gin.Context) {
// 	kingdomName := ctx.Param("kingdom_name")

// 	rulerName := ctx.Param("ruler_name")

// 	rulingID, err := strconv.Atoi(ctx.Param("ruling_id"))
// 	if err != nil {
// 		ctx.String(http.StatusBadRequest, "error parsing rulingID")
// 		return
// 	}

// 	err = a.repo.DeleteKingdomRuler(kingdomName, rulerName, rulingID)
// 	if err != nil {
// 		ctx.String(http.StatusInternalServerError, "error delering kingdom ruler:"+err.Error())
// 		return
// 	}

// 	ctx.String(http.StatusNoContent, "deleting kingdom ruler done successfully")
// }

func (a *Application) checkLogin(ctx *gin.Context) {
	// jwtStr, cookieErr := ctx.Cookie("kingdoms-token")
	// if cookieErr != nil {
	// 	response := responseModels.ResponseDefault{
	// 		Code:    500,
	// 		Status:  "error",
	// 		Message: "error getting cookie",
	// 		Body:    nil,
	// 	}

	// 	ctx.JSON(http.StatusInternalServerError, response)
	// 	return
	// }

	// if !strings.HasPrefix(jwtStr, jwtPrefix) {
	// 	response := responseModels.ResponseDefault{
	// 		Code:    500,
	// 		Status:  "error",
	// 		Message: "error parsing jwt token: no prefix",
	// 		Body:    nil,
	// 	}

	// 	ctx.JSON(http.StatusInternalServerError, response)
	// 	return
	// }

	// jwtStr = jwtStr[len(jwtPrefix):]
	// err := a.redis.CheckJWTInBlacklist(ctx.Request.Context(), jwtStr)
	// if err == nil {
	// 	response := responseModels.ResponseDefault{
	// 		Code:    200,
	// 		Status:  "ok",
	// 		Message: "not authorized: token in black list",
	// 		Body:    nil,
	// 	}

	// 	ctx.JSON(http.StatusOK, response)
	// 	return
	// }

	// token, err := jwt.ParseWithClaims(jwtStr, &serverModels.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
	// 	return []byte(a.config.JWT.Token), nil
	// })

	// if err != nil {
	// 	response := responseModels.ResponseDefault{
	// 		Code:    500,
	// 		Status:  "error",
	// 		Message: "error parsing jwt token: error parsing with claims",
	// 		Body:    nil,
	// 	}

	// 	ctx.JSON(http.StatusInternalServerError, response)
	// 	return
	// }

	// myClaims := token.Claims.(*serverModels.JWTClaims)

	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	assignedRoles := [...]role.Role{role.Unknown, role.Buyer, role.Manager, role.Admin}

	for _, oneOfAssignedRole := range assignedRoles {
		if myClaims.Role == oneOfAssignedRole {
			response := responseModels.ResponseDefault{
				Code:    200,
				Status:  "ok",
				Message: "authorized",
				Body: map[string]interface{}{
					"userRole": oneOfAssignedRole,
					"userName": myClaims.UserName,
				},
			}

			ctx.JSON(http.StatusOK, response)
			return
		}
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "not authorized: no role found",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
	return
}

func (a *Application) login(ctx *gin.Context) {
	cfg := a.config
	request := &serverModels.LoginRequest{}

	err := json.NewDecoder(ctx.Request.Body).Decode(request)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing request params: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	user, err := a.repo.GetUserByName(request.Name)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting user by name: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	if request.Name == user.Name && user.Password == generateHashString(request.Password) {
		token := jwt.NewWithClaims(cfg.JWT.SigningMethod, &serverModels.JWTClaims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(cfg.JWT.ExpiresIn).Unix(),
				IssuedAt:  time.Now().Unix(),
				Issuer:    "bitop-admin",
			},
			UserUUID: uuid.New(),
			Role:     user.Role,
			UserName: user.Name,
		})
		if token == nil {
			response := responseModels.ResponseDefault{
				Code:    500,
				Status:  "error",
				Message: "token is nil",
				Body:    nil,
			}

			ctx.JSON(http.StatusInternalServerError, response)
			return
		}

		strToken, err := token.SignedString([]byte(cfg.JWT.Token))
		if err != nil {
			response := responseModels.ResponseDefault{
				Code:    500,
				Status:  "error",
				Message: "cant create str token",
				Body:    nil,
			}

			ctx.JSON(http.StatusInternalServerError, response)
			return
		}

		ctx.SetCookie("kingdoms-token", jwtPrefix+strToken, int(cfg.JWT.ExpiresIn), "", "", true, true)

		response := responseModels.ResponseDefault{
			Code:    200,
			Status:  "ok",
			Message: "user session starts successfully",
			Body: map[string]interface{}{
				"userRole": user.Role,
				"userName": user.Name,
			},
		}

		ctx.JSON(http.StatusOK, response)
		return
	}

	response := responseModels.ResponseDefault{
		Code:    403,
		Status:  "error",
		Message: "incorrect user data",
		Body:    nil,
	}

	ctx.JSON(http.StatusForbidden, response)
}

func (a *Application) signup(ctx *gin.Context) {
	request := &serverModels.RegisterRequest{}

	err := json.NewDecoder(ctx.Request.Body).Decode(request)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing request params: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	if request.Password == "" {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "password is empty",
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	if request.Name == "" {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "name is empty",
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.Signup(&schema.User{
		UUID:     uuid.New(),
		Role:     role.Buyer,
		Name:     request.Name,
		Password: generateHashString(request.Password),
	})
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error creating user entity: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "user entity created successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func generateHashString(str string) string {
	hasher := sha1.New()
	hasher.Write([]byte(str))

	return hex.EncodeToString(hasher.Sum(nil))
}

func (a *Application) logout(ctx *gin.Context) {
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

	_, err := jwt.ParseWithClaims(jwtStr, &serverModels.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.JWT.Token), nil
	})
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error parsing jwt token: error parsing with claims",
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	err = a.redis.WriteJWTToBlacklist(ctx.Request.Context(), jwtStr, a.config.JWT.ExpiresIn)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error saving in redis black list",
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "user successfully logged out",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}
