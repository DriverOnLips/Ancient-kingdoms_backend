package app

import (
	"context"
	"kingdoms/internal/config"
	"kingdoms/internal/database/connect"
	"kingdoms/internal/database/schema"
	"kingdoms/internal/server/processing"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

	a.r.LoadHTMLGlob("../../templates/*.html")

	a.r.GET("/:Id", a.getKingdom)
	a.r.GET("/", a.getKingdomsFeed)
	a.r.POST("/status/:Id", a.updateKingdomStatus)

	a.r.Run(":8000")

	log.Println("Server is down")
}

func (a *Application) getKingdomsFeed(ctx *gin.Context) {
	kingdomName := ctx.Query("Kingdom_name")

	kingdoms, err := a.repo.GetKingdoms(kingdomName)
	if err != nil {
		log.Println(err)

		ctx.Error(err)
	}

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"Kingdoms":     kingdoms,
		"Kingdom_name": kingdomName,
	})
}

func (a *Application) getKingdom(ctx *gin.Context) {
	kingdomID, err := strconv.Atoi(ctx.Param("Id"))
	if err != nil {
		log.Println(err)

		ctx.Error(err)
	}

	var kingdom schema.Kingdom
	kingdom.Id = uint(kingdomID)

	kingdom, err = a.repo.GetKingdom(kingdom)
	if err != nil {
		ctx.Error(err)
	}

	ctx.HTML(http.StatusOK, "kingdom.html", gin.H{
		"Id":          kingdom.Id,
		"Name":        kingdom.Name,
		"Area":        kingdom.Area,
		"Capital":     kingdom.Capital,
		"Image":       kingdom.Image,
		"Description": kingdom.Description,
		"State":       kingdom.State,
	})
}

func (a *Application) updateKingdomStatus(c *gin.Context) {
	kingdomId, err := strconv.Atoi(c.Param("Id"))
	if err != nil {
		log.Println(err)

		c.Error(err)
	}

	kingdomState := c.PostForm("State")
	if err != nil {
		log.Println(err)

		c.Error(err)
	}

	var newStatus string
	if kingdomState == "Данные подтверждены" {
		newStatus = "Данные утеряны"
	} else {
		newStatus = "Данные подтверждены"
	}

	err = a.repo.UpdateKingdomStatus(kingdomId, newStatus)

	if err != nil {
		log.Println(err)

		c.Error(err)
	}

	c.Redirect(http.StatusFound, "/"+strconv.Itoa(kingdomId))
}
