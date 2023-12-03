package app

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	config "kingdoms/internal/config"
	"kingdoms/internal/database/connect"
	"kingdoms/internal/database/schema"
	role "kingdoms/internal/server/app/userRole"
	requestsModels "kingdoms/internal/server/models/requestModels"
	"kingdoms/internal/server/models/serverModels"
	"kingdoms/internal/server/redis"

	"kingdoms/internal/server/processing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

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

	a.r.GET("kingdoms", a.getKingdoms)
	a.r.GET("kingdom", a.getKingdom)
	a.r.GET("rulers", a.getRulers)
	a.r.GET("ruler", a.getRuler)

	a.r.POST("kingdom/add", a.addKingdom)
	a.r.PUT("kingdom/edit", a.editKingdom)
	a.r.PUT("kingdom/ruler_to_kingdom", a.CreateRulerForKingdom)

	a.r.PUT("ruler/edit", a.editRuler)
	a.r.PUT("ruler/state_change/moderator", a.rulerStateChangeModerator)
	a.r.PUT("ruler/state_change/user", a.rulerStateChangeUser)

	a.r.PUT("kingdom/delete/:kingdom_name", a.deleteKingdom)
	a.r.PUT("kingdom/ruler/:ruler_name", a.deleteRuler)

	a.r.DELETE("kingdom_ruler_delete/:kingdom_name/:ruler_name/:ruling_id", a.deleteKingdomRuler)

	// // никто не имеет доступа
	a.r.Use(a.WithAuthCheck()).GET("login", a.login)
	// // или ниженаписанное значит что доступ имеют менеджер и админ
	// a.r.Use(a.WithAuthCheck(role.Manager, role.Admin)).GET("/ping", a.login)

	// a.r.GET("login", a.checkLogin)
	a.r.POST("login", a.login)
	a.r.POST("signup", a.signup)
	a.r.DELETE("logout", a.logout)

	a.r.Run(":8000")

	log.Println("Server is down")
}

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Body    interface{} `json:"body"`
}

func (a *Application) getKingdoms(ctx *gin.Context) {
	name := ctx.Query("kingdomName")
	ruler := ctx.Query("rulerName")
	state := ctx.Query("state")

	requestBody := requestsModels.GetKingdomsRequest{
		KingdomName: name,
		RulerName:   ruler,
		State:       state,
	}

	kingdoms, err := a.repo.GetKingdoms(requestBody)
	if err != nil {
		response := Response{
			Status:  "error",
			Message: "error getting necessary kingdoms: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)

		return
	}

	response := Response{
		Status:  "ok",
		Message: "kingdoms found",
		Body:    kingdoms,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) getKingdom(ctx *gin.Context) {
	var kingdom schema.Kingdom
	kingdomID, err := strconv.Atoi(ctx.Query("id"))

	fmt.Println(ctx)

	if err != nil {
		response := Response{
			Status:  "error",
			Message: "error getting kingdom ID: " + err.Error(),
			Body:    nil,
		}
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	kingdom.Id = uint(kingdomID)

	kingdom, err = a.repo.GetKingdom(kingdom)
	if err != nil {
		response := Response{
			Status:  "error",
			Message: "error getting necessary kingdom: " + err.Error(),
			Body:    nil,
		}
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response := Response{
		Status:  "ok",
		Message: "kingdom found",
		Body:    kingdom,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) getRulers(ctx *gin.Context) {
	var requestBody requestsModels.GetRulersRequest
	if err := ctx.BindJSON(&requestBody); err != nil {
		ctx.String(http.StatusBadRequest, "error parsing request body:"+err.Error())
		return
	}

	rulers, err := a.repo.GetRulers(requestBody)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error getting rulers:"+err.Error())
		return
	}

	ctx.JSON(http.StatusFound, rulers)
}

func (a *Application) getRuler(ctx *gin.Context) {
	var ruler schema.Ruler
	if err := ctx.BindJSON(&ruler); err != nil {
		ctx.String(http.StatusBadRequest, "error parsing ruler:"+err.Error())
		return
	}

	necessaryRuler, err := a.repo.GetRuler(ruler)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error getting ruler:"+err.Error())
		return
	}
	if necessaryRuler == (schema.Ruler{}) {
		ctx.String(http.StatusNotFound, "no necessary ruler")
		return
	}

	ctx.JSON(http.StatusFound, necessaryRuler)
}

func (a *Application) addKingdom(ctx *gin.Context) {
	var kingdom schema.Kingdom
	if err := ctx.BindJSON(&kingdom); err != nil {
		ctx.String(http.StatusBadRequest, "error parsing kingdom:"+err.Error())
		return
	}

	err := a.repo.CreateKingdom(kingdom)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error creating kingdom:"+err.Error())
		return
	}

	ctx.String(http.StatusCreated, "creating kingdom done successfully")
}

func (a *Application) editKingdom(ctx *gin.Context) {
	var kingdom schema.Kingdom
	if err := ctx.BindJSON(&kingdom); err != nil {
		ctx.String(http.StatusBadRequest, "error parsing ruler")
		return
	}

	err := a.repo.EditKingdom(kingdom)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error editing kingdom:"+err.Error())
		return
	}

	ctx.JSON(http.StatusNoContent, kingdom)
}

func (a *Application) CreateRulerForKingdom(ctx *gin.Context) {
	var requestBody requestsModels.CreateRulerForKingdomRequest
	if err := ctx.BindJSON(&requestBody); err != nil {
		ctx.String(http.StatusBadRequest, "error parsing kingdom:"+err.Error())
		return
	}

	err := a.repo.CreateRulerForKingdom(requestBody)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error ruler for kingdom additing:"+err.Error())
		return
	}

	ctx.String(http.StatusNoContent, "additing done successfully")
}

func (a *Application) editRuler(ctx *gin.Context) {
	var ruler schema.Ruler
	if err := ctx.BindJSON(&ruler); err != nil {
		ctx.String(http.StatusBadRequest, "error parsing ruler:"+err.Error())
		return
	}

	err := a.repo.EditRuler(ruler)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error editing ruler:"+err.Error())
		return
	}

	ctx.String(http.StatusNoContent, "edditing ruler done successfully")
}

func (a *Application) rulerStateChangeModerator(ctx *gin.Context) {
	var requestBody requestsModels.RulerStateChangeRequest
	if err := ctx.BindJSON(&requestBody); err != nil {
		ctx.String(http.StatusBadRequest, "error parsing request body:"+err.Error())
		return
	}

	userRole, err := a.repo.GetUserRole(requestBody.User)
	if err != nil {
		ctx.String(http.StatusBadRequest, "error getting user role:"+err.Error())
		return
	}
	if userRole != "admin" {
		ctx.String(http.StatusUnauthorized, "no enouth rules for executing this operation")
		return
	}

	err = a.repo.RulerStateChange(requestBody.ID, requestBody.State)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error ruler state changing:"+err.Error())
		return
	}

	ctx.String(http.StatusNoContent, "ruler state changing done successfully")
}

func (a *Application) rulerStateChangeUser(ctx *gin.Context) {
	var requestBody requestsModels.RulerStateChangeRequest
	if err := ctx.BindJSON(&requestBody); err != nil {
		ctx.String(http.StatusBadRequest, "error parsing request body:"+err.Error())
		return
	}

	userRole, err := a.repo.GetUserRole(requestBody.User)
	if err != nil {
		ctx.String(http.StatusBadRequest, "error getting user role:"+err.Error())
		return
	}
	if userRole != "user" && userRole != "admin" {
		ctx.String(http.StatusUnauthorized, "no enouth rules for executing this operation")
		return
	}

	err = a.repo.RulerStateChange(requestBody.ID, requestBody.State)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error ruler state changing:"+err.Error())
		return
	}

	ctx.String(http.StatusNoContent, "ruler state changing done successfully")
}

func (a *Application) deleteKingdom(ctx *gin.Context) {
	kingdomName := ctx.Param("kingdom_name")

	err := a.repo.DeleteKingdom(kingdomName)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error deleting kingdom:"+err.Error())
		return
	}

	ctx.String(http.StatusNoContent, "deleting kingdom done successfully")
}

func (a *Application) deleteRuler(ctx *gin.Context) {
	rulerName := ctx.Param("ruler_name")

	err := a.repo.DeleteRuler(rulerName)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error deleting ruler:"+err.Error())
		return
	}

	ctx.String(http.StatusNoContent, "deleting ruler done successfully")
}

func (a *Application) deleteKingdomRuler(ctx *gin.Context) {
	kingdomName := ctx.Param("kingdom_name")

	rulerName := ctx.Param("ruler_name")

	rulingID, err := strconv.Atoi(ctx.Param("ruling_id"))
	if err != nil {
		ctx.String(http.StatusBadRequest, "error parsing rulingID")
		return
	}

	err = a.repo.DeleteKingdomRuler(kingdomName, rulerName, rulingID)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error delering kingdom ruler:"+err.Error())
		return
	}

	ctx.String(http.StatusNoContent, "deleting kingdom ruler done successfully")
}

func (a *Application) checkLogin(ctx *gin.Context) {

}

func (a *Application) login(gCtx *gin.Context) {
	cfg := a.config
	req := &loginReq{}

	err := json.NewDecoder(gCtx.Request.Body).Decode(req)
	if err != nil {
		gCtx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	user, err := a.repo.GetUserByName(req.Login)
	if err != nil {
		gCtx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if req.Login == user.Name && user.Pass == generateHashString(req.Password) {
		// значит проверка пройдена
		// генерируем ему jwt
		token := jwt.NewWithClaims(cfg.JWT.SigningMethod, &serverModels.JWTClaims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(cfg.JWT.ExpiresIn).Unix(),
				IssuedAt:  time.Now().Unix(),
				Issuer:    "bitop-admin",
			},
			UserUUID: uuid.New(), // test uuid
			Role:     user.Role,
		})
		if token == nil {
			gCtx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("token is nil"))
			return
		}

		strToken, err := token.SignedString([]byte(cfg.JWT.Token))
		if err != nil {
			gCtx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("cant create str token"))
			return
		}

		gCtx.JSON(http.StatusOK, serverModels.LoginResponce{
			ExpiresIn:   cfg.JWT.ExpiresIn,
			AccessToken: strToken,
			TokenType:   "Bearer",
		})
	}

	gCtx.AbortWithStatus(http.StatusForbidden) // отдаем 403 ответ в знак того что доступ запрещен
}

func (a *Application) signup(gCtx *gin.Context) {
	req := &registerReq{}

	err := json.NewDecoder(gCtx.Request.Body).Decode(req)
	if err != nil {
		gCtx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if req.Pass == "" {
		gCtx.AbortWithError(http.StatusBadRequest, fmt.Errorf("pass is empty"))
		return
	}

	if req.Name == "" {
		gCtx.AbortWithError(http.StatusBadRequest, fmt.Errorf("name is empty"))
		return
	}

	err = a.repo.Singup(&schema.User{
		UUID: uuid.New(),
		Role: role.Buyer,
		Name: req.Name,
		Pass: generateHashString(req.Pass), // пароли делаем в хешированном виде и далее будем сравнивать хеши, чтобы их не угнали с базой вместе
	})
	if err != nil {
		gCtx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	gCtx.JSON(http.StatusOK, &registerResp{
		Ok: true,
	})
}

func generateHashString(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func (a *Application) logout(ctx *gin.Context) {
	// получаем заголовок
	jwtStr := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(jwtStr, jwtPrefix) { // если нет префикса то нас дурят!
		gCtx.AbortWithStatus(http.StatusBadRequest) // отдаем что нет доступа

		return // завершаем обработку
	}

	// отрезаем префикс
	jwtStr = jwtStr[len(jwtPrefix):]

	_, err := jwt.ParseWithClaims(jwtStr, &ds.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.JWT.Token), nil
	})
	if err != nil {
		gCtx.AbortWithError(http.StatusBadRequest, err)
		log.Println(err)

		return
	}

	// сохраняем в блеклист редиса
	err = a.redis.WriteJWTToBlacklist(gCtx.Request.Context(), jwtStr, a.config.JWT.ExpiresIn)
	if err != nil {
		gCtx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	gCtx.Status(http.StatusOK)
}

// func (a *Application) loadKingdoms(c *gin.Context) {
// 	kingdomName := c.Query("kingdom_name")

// 	if kingdomName == "" {
// 		allKingdoms, err := a.repo.GetAllKingdoms()

// 		if err != nil {
// 			log.Println(err)
// 			c.Error(err)
// 		}

// 		c.HTML(http.StatusOK, "index.html", gin.H{
// 			"kingdoms": allKingdoms,
// 		})
// 	} else {
// 		foundKingdoms, err := a.repo.SearchKingdoms(kingdomName)

// 		if err != nil {
// 			c.Error(err)
// 			return
// 		}

// 		c.HTML(http.StatusOK, "index.html", gin.H{
// 			"kingdoms":   foundKingdoms,
// 			"searchText": kingdomName,
// 		})
// 	}
// }

// func (a *Application) loadKingdom(c *gin.Context) {
// 	kingdomName := c.Param("kingdom_name")

// 	if kingdomName == "favicon.ico" {
// 		return
// 	}

// 	kingdom, err := a.repo.GetKingdomByName(kingdomName)

// 	if err != nil {
// 		c.Error(err)
// 		return
// 	}

// 	c.HTML(http.StatusOK, "kingdom.html", gin.H{
// 		"Name":        kingdom.Name,
// 		"Image":       kingdom.Image,
// 		"Description": kingdom.Description,
// 		"Capital":     kingdom.Capital,
// 		"Area":        kingdom.Area,
// 		"State":       kingdom.State,
// 	})
// }

// func (a *Application) loadKingdomChangeVisibility(c *gin.Context) {
// 	kingdomName := c.Param("kingdom_name")
// 	err := a.repo.ChangeKingdomVisibility(kingdomName)

// 	if err != nil {
// 		c.Error(err)
// 	}

// 	c.Redirect(http.StatusFound, "/"+kingdomName)
// }
