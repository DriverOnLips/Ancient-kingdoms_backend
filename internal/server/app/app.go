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

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag/example/basic/docs"

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
	"gorm.io/datatypes"
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

	// swagger
	docs.SwaggerInfo.BasePath = "/"
	a.r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	a.r.GET("kingdoms", a.getKingdomsFeed)
	a.r.GET("kingdom", a.getKingdom)
	a.r.GET("applications", a.getApplications)
	a.r.GET("application/with_kingdoms", a.getApplicationWithKingdoms)
	a.r.GET("applications/all", a.getAllApplications)

	a.r.POST("kingdom/create", a.createKingdom)
	a.r.POST("application/create", a.createApplication)

	a.r.PUT("kingdom/update", a.updateKingdom)
	a.r.PUT("kingdom/update/status", a.updateKingdomStatus)
	a.r.PUT("application/status", a.updateApplicationStatus)
	a.r.PUT("application/update", a.updateApplication)
	a.r.PUT("application/add_kingdom", a.addKingdomToApplication)
	a.r.PUT("application/update_kingdom", a.updateKingdomFromApplication)

	a.r.DELETE("application/delete_kingdom", a.deleteKingdomFromApplication)
	a.r.DELETE("application/delete", a.deleteApplication)

	a.r.GET("login", a.checkLogin)
	a.r.POST("login", a.login)
	a.r.POST("signup", a.signup)
	a.r.DELETE("logout", a.logout)

	a.r.Run(":8000")

	log.Println("Server is down")
}

// @Summary Получить все княжества
// @Description  Возвращает все княжества
// @Tags Княжество
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param Kingdom_name query string false "Название княжества"
// @Router /kingdoms [get]
func (a *Application) getKingdomsFeed(ctx *gin.Context) {
	kingdomName := ctx.Query("Kingdom_name") // TODO окно для пагинации

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

// @Summary Получить данные о княжестве
// @Description  Возвращает все поля требуемого княжества
// @Tags Княжество
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param Id query string false "Id княжества"
// @Router /kingdom [get]
func (a *Application) getKingdom(ctx *gin.Context) {
	kingdomID, err := strconv.Atoi(ctx.Query("Id"))
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

// @Summary Добавить княжество в БД
// @Description  Создает новое княжество по переданным параметрам
// @Tags Княжество
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param kingdomToCreate body schema.Kingdom true "Княжество"
// @Router /kingdom/create [post]
func (a *Application) createKingdom(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	_, err := a.repo.GetUserByName(myClaims.Name)
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

	var kingdomToCreate schema.Kingdom
	if err := ctx.BindJSON(&kingdomToCreate); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing kingdom:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.CreateKingdom(kingdomToCreate)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error creating kingdom: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom created successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Обновляет княжество
// @Description  Обновляет переданные параметры у княжества
// @Tags Княжество
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param kingdomToUpdate body schema.Kingdom true "Княжество"
// @Router /kingdom/update [put]
func (a *Application) updateKingdom(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	_, err := a.repo.GetUserByName(myClaims.Name)
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

	var kingdomToUpdate schema.Kingdom
	if err := ctx.BindJSON(&kingdomToUpdate); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing kingdom:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.UpdateKingdom(kingdomToUpdate)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error updating kingdom: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom updated successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Обновляет статус княжества
// @Description  Обновляет статус у княжества
// @Tags Княжество
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param kingdomToUpdate body processing.KingdomToUpdate true "Тело запроса"
// @Router /kingdom/update/status [put]
func (a *Application) updateKingdomStatus(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	_, err := a.repo.GetUserByName(myClaims.Name)
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

	var kingdomToUpdate processing.KingdomToUpdate
	if err := ctx.BindJSON(&kingdomToUpdate); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing kingdom:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.UpdateKingdomStatus(kingdomToUpdate)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error updating status kingdom: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom status updated successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Получить запись
// @Description  Возвращает запись, удовлетворяющую входным параметрам
// @Tags Записи
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param Id query string false "Id записи"
// @Router /applications [get]
func (a *Application) getApplications(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.Name)
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

	applicationId := ctx.Query("Id")

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

// @Summary Получить все записи
// @Description  Возвращает все записи, удовлетворяющие фильтрам
// @Tags Записи
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param Status query string false "Статус записи"
// @Param From query string false "Дата начала" Format(date-time)
// @Param To query string false "Дата окончания" Format(date-time)
// @Router /applications/all [get]
func (a *Application) getAllApplications(ctx *gin.Context) {
	fromStr := strings.Split(ctx.Query("From"), "T")[0]
	toStr := strings.Split(ctx.Query("To"), "T")[0]

	var from time.Time
	var to time.Time
	var err error

	if fromStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			response := responseModels.ResponseDefault{
				Code:    400,
				Status:  "error",
				Message: "error parsing dateFrom: " + err.Error(),
				Body:    nil,
			}

			ctx.JSON(http.StatusBadRequest, response)
			return
		}
	}

	if toStr != "" {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			response := responseModels.ResponseDefault{
				Code:    400,
				Status:  "error",
				Message: "error parsing dateTo: " + err.Error(),
				Body:    nil,
			}

			ctx.JSON(http.StatusBadRequest, response)
			return
		}
	}

	params := processing.StructGetAllApplications{
		Status: ctx.Query("Status"),
		From:   datatypes.Date(from),
		To:     datatypes.Date(to),
	}

	applications, err := a.repo.GetAllApplications(params)
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
		Message: "applications found",
		Body:    applications,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Получить запись с княжествами
// @Description  Возвращает определенную запись и все княжества, входящие в нее
// @Tags Записи
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param Id query string true "Id записи"
// @Router /applications/with_kingdoms [get]
func (a *Application) getApplicationWithKingdoms(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.Name)
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

	applicationId := ctx.Query("Id")
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

	application, err := a.repo.GetApplicationWithKingdoms(*user, applicationId)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting necessary kingdoms from application: " + err.Error(),
			Body:    nil,
		}
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdoms from application found",
		Body:    application,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Добавить запись в БД
// @Description  Создает новую запись и добавляет в нее переданное княжество
// @Tags Записи
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param applicationToAdd query processing.KingdomAddToApplication true "Тело запроса"
// @Router /application/create [post]
func (a *Application) createApplication(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.Name)
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

	var applicationToAdd processing.KingdomAddToApplication
	if err := ctx.BindJSON(&applicationToAdd); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing application:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	application, err := a.repo.CreateApplication(*user)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error creating application: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	applicationToAdd.ApplicationId = application.Id

	err = a.repo.AddKingdomToApplication(*user, applicationToAdd)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error adding kingdom to application: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	applicationWithKingdoms, err := a.repo.GetApplicationWithKingdoms(*user,
		strconv.Itoa(int(applicationToAdd.ApplicationId)))

	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting necessary kingdoms from application: " + err.Error(),
			Body:    nil,
		}
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdoms from application found",
		Body:    applicationWithKingdoms,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Обновить статус записи
// @Description  Обновляет статус записи на переданный
// @Tags Записи
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param applicationToUpdate query processing.ApplicationToUpdate true "Запись"
// @Router /application/status [put]
func (a *Application) updateApplicationStatus(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.Name)
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

	var applicationToUpdate processing.ApplicationToUpdate
	if err := ctx.BindJSON(&applicationToUpdate); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing application:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.UpdateApplicationStatus(*user, applicationToUpdate)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error updating application status: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "appliction status updated successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Обновить запись
// @Description  Обновляет запись
// @Tags Записи
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param applicationToUpdate query schema.RulerApplication true "Запись"
// @Router /application/update [put]
func (a *Application) updateApplication(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.Name)
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

	var applicationToUpdate schema.RulerApplication
	if err := ctx.BindJSON(&applicationToUpdate); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing application:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.UpdateApplication(*user, applicationToUpdate)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error updating application ruler: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "appliction ruler updated successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Добавить княжество в запись
// @Description  Добавляет княжество в запись-черновик
// @Tags Запись
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param kingdomAddToApplication query processing.KingdomAddToApplication true "Тело запроса"
// @Router /application/add_kingdom [put]
func (a *Application) addKingdomToApplication(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.Name)
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

	var kingdomAddToApplication processing.KingdomAddToApplication
	if err := ctx.BindJSON(&kingdomAddToApplication); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing kingdom or application:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.AddKingdomToApplication(*user, kingdomAddToApplication)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error adding kingdom to application: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom added to application successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Обновить княжество в записи
// @Description  Изменяет сроки правления княжеством в записи
// @Tags Запись
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param updateKingdomFromApplication query processing.KingdomAddToApplication true "Тело запроса"
// @Router /application/update_kingdom [put]
func (a *Application) updateKingdomFromApplication(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.Name)
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

	var updateKingdomFromApplication processing.KingdomAddToApplication
	if err := ctx.BindJSON(&updateKingdomFromApplication); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing kingdom or application:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.UpdateKingdomFromApplication(*user, updateKingdomFromApplication)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error adding kingdom to application: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom added to application successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Удалить княжество из записи
// @Description  Удаляет княжество из записи-черновика
// @Tags Запись
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param kingdomToDeleteFromApplication query processing.DeleteKingdomFromApplication true "Тело запроса"
// @Router /application/delete_kingdom [delete]
func (a *Application) deleteKingdomFromApplication(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.Name)
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

	var kingdomToDeleteFromApplication processing.DeleteKingdomFromApplication
	if err := ctx.BindJSON(&kingdomToDeleteFromApplication); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing kingdom or application:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.DeleteKingdomFromApplication(*user, kingdomToDeleteFromApplication)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error deleting kingdom from application: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom deleted from application successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Удалить запись
// @Description  Удаляет запись
// @Tags Запись
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param applicatinToDelete query schema.RulerApplication true "Запись"
// @Router /application/delete [delete]
func (a *Application) deleteApplication(ctx *gin.Context) {
	myClaims, response := a.repo.FoundUserFromHeader(ctx, a.redis, a.config)
	if response != (responseModels.ResponseDefault{}) {
		ctx.JSON(response.Code, response)
		return
	}

	user, err := a.repo.GetUserByName(myClaims.Name)
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

	var applicatinToDelete schema.RulerApplication
	if err := ctx.BindJSON(&applicatinToDelete); err != nil {
		response := responseModels.ResponseDefault{
			Code:    400,
			Status:  "error",
			Message: "error parsing application:" + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	err = a.repo.DeleteApplication(*user, applicatinToDelete)
	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error deleting application: " + err.Error(),
			Body:    nil,
		}

		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response = responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "application deleted successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Проверить сессию
// @Description Возвращает jwt токен
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200 {} json
// @Router /login [get]
func (a *Application) checkLogin(ctx *gin.Context) {
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
					"Id":   myClaims.Id,
					"Name": myClaims.Name,
					"Role": oneOfAssignedRole,
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

// @Summary Войти в систему
// @Description Возвращает jwt токен
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200 {} json
// @Param request body serverModels.LoginRequest true "Тело запроса"
// @Router /login [post]
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
			Id:       user.Id,
			Name:     user.Name,
			Role:     user.Role,
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
				"Id":   user.Id,
				"Role": user.Role,
				"Name": user.Name,
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

// @Summary Регистрация в системе
// @Description Создает пользователя в БД
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200 {} json
// @Param request body serverModels.RegisterRequest true "Тело запроса"
// @Router /signup [post]
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

// @Summary Выход из системы
// @Description Удаляет сессию пользователя
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200 {} json
// @Router /logout [delete]
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
			Message: "error parsing jwt token: error parsing with claims: " + err.Error(),
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
