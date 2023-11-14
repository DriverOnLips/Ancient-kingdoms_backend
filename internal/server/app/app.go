package app

import (
	"log"
	"net/http"
	"strconv"

	"kingdoms/internal/database/connect"
	requests "kingdoms/internal/database/requestModel"
	"kingdoms/internal/database/schema"
	"kingdoms/internal/server/processing"

	"github.com/gin-gonic/gin"
)

type Application struct {
	repo processing.Repository
	r    *gin.Engine
}

func New() Application {
	app := Application{}

	repo, _ := processing.New(connect.FromEnv())

	app.repo = *repo

	return app

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

	a.r.GET("login", a.checkLogin)
	a.r.PUT("login", a.login)
	a.r.PUT("signup", a.signup)
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
	ruler := ctx.Query("ruler")
	state := ctx.Query("state")

	requestBody := requests.GetKingdomsRequest{
		Ruler: ruler,
		State: state,
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

	// kingdoms[0].Image = "http://localhost:8000/internal/database/store/img/kingdoms/Chernigov.png"

	response := Response{
		Status:  "ok",
		Message: "kingdoms found",
		Body:    kingdoms,
	}
	ctx.JSON(http.StatusOK, response)
}

func (a *Application) getKingdom(ctx *gin.Context) {
	var kingdom schema.Kingdom
	if err := ctx.BindJSON(&kingdom); err != nil {
		ctx.String(http.StatusBadRequest, "error parsing kingdom:"+err.Error())
		return
	}

	necessaryKingdom, err := a.repo.GetKingdom(kingdom)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error getting kingdom:"+err.Error())
		return
	}
	if necessaryKingdom == (schema.Kingdom{}) {
		ctx.String(http.StatusNotFound, "no necessary kingdom")
		return
	}

	ctx.JSON(http.StatusFound, necessaryKingdom)
}

func (a *Application) getRulers(ctx *gin.Context) {
	var requestBody requests.GetRulersRequest
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
	var requestBody requests.CreateRulerForKingdomRequest
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
	var requestBody requests.RulerStateChangeRequest
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
	var requestBody requests.RulerStateChangeRequest
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

func (a *Application) login(ctx *gin.Context) {

}

func (a *Application) signup(ctx *gin.Context) {

}

func (a *Application) logout(ctx *gin.Context) {

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
