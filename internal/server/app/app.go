package app

import (
	"context"
	"kingdoms/internal/config"
	"kingdoms/internal/database/connect"
	"kingdoms/internal/database/schema"
	"kingdoms/internal/server/models/responseModels"
	"kingdoms/internal/server/processing"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type Application struct {
	config *config.Config
	repo   *processing.Repository
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

	return &Application{
		config: cfg,
		repo:   repo,
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

	a.r.Run(":8000")

	log.Println("Server is down")
}

func (a *Application) getKingdomsFeed(ctx *gin.Context) {
	kingdomName := ctx.Query("Kingdom_name")

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

func (a *Application) createKingdom(ctx *gin.Context) {
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

	err := a.repo.CreateKingdom(kingdomToCreate)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom created successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) updateKingdom(ctx *gin.Context) {
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

	err := a.repo.UpdateKingdom(kingdomToUpdate)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom updated successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) updateKingdomStatus(ctx *gin.Context) {
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

	err := a.repo.UpdateKingdomStatus(kingdomToUpdate)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom status updated successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) getApplications(ctx *gin.Context) {
	applicationId := ctx.Query("Id")

	applications, err := a.repo.GetApplications(applicationId)
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

func (a *Application) getApplicationWithKingdoms(ctx *gin.Context) {
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

	application, err := a.repo.GetApplicationWithKingdoms(applicationId)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdoms from application found",
		Body:    application,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) createApplication(ctx *gin.Context) {
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

	application, err := a.repo.CreateApplication()
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

	err = a.repo.AddKingdomToApplication(applicationToAdd)
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

	applicationWithKingdoms, err := a.repo.GetApplicationWithKingdoms(
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdoms from application found",
		Body:    applicationWithKingdoms,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) updateApplicationStatus(ctx *gin.Context) {
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

	err := a.repo.UpdateApplicationStatus(applicationToUpdate)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "appliction status updated successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) updateApplication(ctx *gin.Context) {
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

	err := a.repo.UpdateApplication(applicationToUpdate)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "appliction ruler updated successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) addKingdomToApplication(ctx *gin.Context) {
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

	err := a.repo.AddKingdomToApplication(kingdomAddToApplication)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom added to application successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) updateKingdomFromApplication(ctx *gin.Context) {
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

	err := a.repo.UpdateKingdomFromApplication(updateKingdomFromApplication)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom added to application successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) deleteKingdomFromApplication(ctx *gin.Context) {
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

	err := a.repo.DeleteKingdomFromApplication(kingdomToDeleteFromApplication)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "kingdom deleted from application successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}

func (a *Application) deleteApplication(ctx *gin.Context) {
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

	err := a.repo.DeleteApplication(applicatinToDelete)
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

	response := responseModels.ResponseDefault{
		Code:    200,
		Status:  "ok",
		Message: "application deleted successfully",
		Body:    nil,
	}

	ctx.JSON(http.StatusOK, response)
}
