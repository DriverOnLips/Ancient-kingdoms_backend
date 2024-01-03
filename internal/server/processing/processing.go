package processing

import (
	"errors"
	"fmt"
	"kingdoms/internal/config"
	"kingdoms/internal/database/schema"
	"kingdoms/internal/server/models/responseModels"
	"kingdoms/internal/server/models/serverModels"
	"kingdoms/internal/server/redis"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const jwtPrefix = "Bearer"

type Repository struct {
	db *gorm.DB
}

func New(connect string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(connect), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Repository{
		db: db,
	}, nil
}

func (r *Repository) FoundUserFromHeader(ctx *gin.Context, redis *redis.Client, config *config.Config) (*serverModels.JWTClaims, responseModels.ResponseDefault) {
	jwtStr, cookieErr := ctx.Cookie("kingdoms-token")
	if cookieErr != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error getting cookie",
			Body:    nil,
		}

		return nil, response
	}

	if !strings.HasPrefix(jwtStr, jwtPrefix) {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error parsing jwt token: no prefix",
			Body:    nil,
		}

		return nil, response
	}

	jwtStr = jwtStr[len(jwtPrefix):]
	err := redis.CheckJWTInBlacklist(ctx.Request.Context(), jwtStr)
	if err == nil {
		response := responseModels.ResponseDefault{
			Code:    403,
			Status:  "error",
			Message: "not authorized: token in black list",
			Body:    nil,
		}

		return nil, response
	}

	token, err := jwt.ParseWithClaims(jwtStr, &serverModels.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWT.Token), nil
	})

	if err != nil {
		response := responseModels.ResponseDefault{
			Code:    500,
			Status:  "error",
			Message: "error parsing jwt token: error parsing with claims: " + err.Error(),
			Body:    nil,
		}

		return nil, response
	}

	return token.Claims.(*serverModels.JWTClaims), responseModels.ResponseDefault{}
}

func (r *Repository) AsyncGetApplication(applicationId string) (AsyncStructApplication, error) {
	var applicationToReturn AsyncStructApplication

	var tx *gorm.DB = r.db
	err := tx.Table("ruler_applications").
		Select("id, 'check'").
		Where("id = ?", applicationId).
		Scan(&applicationToReturn).Error
	if err != nil {
		return AsyncStructApplication{}, err
	}

	if applicationToReturn == (AsyncStructApplication{}) {
		return AsyncStructApplication{}, errors.New("application not found")
	}

	return applicationToReturn, nil
}

func (r *Repository) AsyncPutApplicationInfo(applicationToPut AsyncStructApplication) error {
	var tx *gorm.DB = r.db

	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", applicationToPut.Id).
		Update("check", applicationToPut.Check).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetKingdoms(kingdomName string) ([]schema.Kingdom, error) {
	kingdomsToReturn := []schema.Kingdom{}

	var tx *gorm.DB = r.db

	err := tx.Where("name LIKE " + "'%" + kingdomName + "%'").Find(&kingdomsToReturn).Error
	if err != nil {
		return []schema.Kingdom{}, err
	}

	if len(kingdomsToReturn) == 0 {
		return []schema.Kingdom{}, errors.New("no necessary kingdoms found")
	}

	return kingdomsToReturn, nil
}

func (r *Repository) GetKingdom(kingdom schema.Kingdom) (schema.Kingdom, error) {
	var kingdomToReturn schema.Kingdom

	err := r.db.Where(kingdom).First(&kingdomToReturn).Error
	if err != nil {
		return schema.Kingdom{}, err
	} else {
		return kingdomToReturn, nil
	}
}

func (r *Repository) GetApplications(user schema.User, applicationId string) ([]schema.RulerApplication, error) {
	var applicationsToReturn []schema.RulerApplication

	var tx *gorm.DB = r.db

	if applicationId == "" {
		err := tx.Where("creator_refer = ? and state != 'Удалена'", user.Id).
			Order("id").
			Find(&applicationsToReturn).Error
		if err != nil {
			return []schema.RulerApplication{}, err
		}

		if len(applicationsToReturn) == 0 {
			return []schema.RulerApplication{}, errors.New("no necessary ruler applications found")
		}

		return applicationsToReturn, nil
	}

	err := tx.Where("id = ?", applicationId).Find(&applicationsToReturn).Error
	if err != nil {
		return []schema.RulerApplication{}, err
	}

	if len(applicationsToReturn) == 0 {
		return []schema.RulerApplication{}, errors.New("no necessary ruler applications found")
	}

	return applicationsToReturn, nil
}

func (r *Repository) GetApplicationWithKingdoms(user schema.User, applicationId string) (StructApplicationWithKingdoms, error) {
	nestedApplication, err := r.GetApplications(user, applicationId)
	if err != nil {
		return StructApplicationWithKingdoms{}, err
	}

	var applicationToReturn StructApplicationWithKingdoms
	applicationToReturn.Application = nestedApplication[0]
	var kingdom2Application []schema.Kingdom2Application

	var tx *gorm.DB = r.db

	err = tx.Where("application_refer = ?", applicationId).Find(&kingdom2Application).Error
	if err != nil {
		return StructApplicationWithKingdoms{}, err
	}

	if len(kingdom2Application) == 0 {
		// return StructApplicationWithKingdoms{}, errors.New("kingdom2Application from this application not found")
		return applicationToReturn, nil
	}

	for i := 0; i < len(kingdom2Application); i++ {
		var nestedKingdom schema.Kingdom
		err = tx.Where("id = ?", kingdom2Application[i].KingdomRefer).Find(&nestedKingdom).Error
		if err != nil {
			return StructApplicationWithKingdoms{}, err
		}

		var kingdomFromApplication KingdomFromApplication
		kingdomFromApplication.Kingdom = nestedKingdom
		kingdomFromApplication.From = kingdom2Application[i].From
		kingdomFromApplication.To = kingdom2Application[i].To

		applicationToReturn.Kingdoms = append(applicationToReturn.Kingdoms, kingdomFromApplication)
	}

	return applicationToReturn, nil
}

func (r *Repository) CreateApplication(user schema.User) (schema.RulerApplication, error) {
	application := schema.RulerApplication{
		Creator:    user,
		State:      "В разработке",
		DateCreate: datatypes.Date(time.Now()),
	}

	var tx *gorm.DB = r.db

	err := tx.Create(&application).Error
	if err != nil {
		return schema.RulerApplication{}, err
	}

	return application, nil
}

func (r *Repository) UpdateApplicationStatus(user schema.User, applicationToUpdate ApplicationToUpdate) (AsyncStructApplication, error) {
	var tx *gorm.DB = r.db

	var app schema.RulerApplication
	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", applicationToUpdate.Id).
		Where("creator_refer = ?", user.Id).
		First(&app).Error
	if err != nil {
		return AsyncStructApplication{}, err
	}
	if app == (schema.RulerApplication{}) {
		return AsyncStructApplication{}, errors.New("no necessary application found")
	}

	switch applicationToUpdate.State {
	case "На рассмотрении":
		err = tx.Model(&schema.RulerApplication{}).
			Where("id = ?", applicationToUpdate.Id).
			Where("creator_refer = ?", user.Id).
			Updates(map[string]interface{}{
				"state":     applicationToUpdate.State,
				"date_send": datatypes.Date(time.Now()),
			}).Error
		if err != nil {
			return AsyncStructApplication{}, err
		}

		break

	case "Одобрена":
		err = tx.Model(&schema.RulerApplication{}).
			Where("id = ?", applicationToUpdate.Id).
			Where("creator_refer = ?", user.Id).
			Updates(map[string]interface{}{
				"state":         applicationToUpdate.State,
				"date_complete": datatypes.Date(time.Now()),
			}).Error
		if err != nil {
			return AsyncStructApplication{}, err
		}
		break

	case "Отклонена":
		err = tx.Model(&schema.RulerApplication{}).
			Where("id = ?", applicationToUpdate.Id).
			Where("creator_refer = ?", user.Id).
			Updates(map[string]interface{}{
				"state":         applicationToUpdate.State,
				"date_complete": datatypes.Date(time.Now()),
			}).Error
		if err != nil {
			return AsyncStructApplication{}, err
		}
		break

	case "Удалена":
		err = tx.Model(&schema.RulerApplication{}).
			Where("id = ?", applicationToUpdate.Id).
			Where("creator_refer = ?", user.Id).
			Update("state", applicationToUpdate.State).Error
		if err != nil {
			return AsyncStructApplication{}, err
		}
		break

	default:
		break
	}

	var applicationToReturn AsyncStructApplication
	err = tx.Table("ruler_applications").
		Select("id, 'check'").
		Where("id = ?", applicationToUpdate.Id).
		Where("creator_refer = ?", user.Id).
		Scan(&applicationToReturn).Error

	err = tx.Model(&schema.RulerApplication{}).
		Where("id = ?", applicationToUpdate.Id).
		Where("creator_refer = ?", user.Id).
		First(&applicationToReturn).Error

	return applicationToReturn, nil
}

func (r *Repository) UpdateApplication(user schema.User,
	applicationToUpdate schema.RulerApplication) error {

	var tx *gorm.DB = r.db

	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", applicationToUpdate.Id).
		Update("ruler", applicationToUpdate.Ruler).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) AddKingdomToApplication(user schema.User, kingdomAddToApplication KingdomAddToApplication) error {
	var tx *gorm.DB = r.db

	var app schema.RulerApplication

	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", kingdomAddToApplication.ApplicationId).
		Where("creator_refer = ?", user.Id).
		First(&app).Error
	if err != nil {
		return err
	}
	if app == (schema.RulerApplication{}) {
		return errors.New("no necessary application found")
	}

	var kingdom2Application = schema.Kingdom2Application{
		ApplicationRefer: int(kingdomAddToApplication.ApplicationId),
		KingdomRefer:     int(kingdomAddToApplication.KingdomId),
		From:             kingdomAddToApplication.From,
		To:               kingdomAddToApplication.To,
	}

	err = r.db.Create(&kingdom2Application).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) UpdateKingdomFromApplication(user schema.User, kingdomAddToApplication KingdomAddToApplication) error {
	var tx *gorm.DB = r.db

	var app schema.RulerApplication

	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", kingdomAddToApplication.ApplicationId).
		Where("creator_refer = ?", user.Id).
		First(&app).Error
	if err != nil {
		return err
	}
	if app == (schema.RulerApplication{}) {
		return errors.New("no necessary application found")
	}

	var kingdom2Application = schema.Kingdom2Application{
		ApplicationRefer: int(kingdomAddToApplication.ApplicationId),
		KingdomRefer:     int(kingdomAddToApplication.KingdomId),
		From:             kingdomAddToApplication.From,
		To:               kingdomAddToApplication.To,
	}

	err = r.db.Model(&schema.Kingdom2Application{}).
		Where("application_refer = ? AND kingdom_refer = ?",
			kingdom2Application.ApplicationRefer, kingdom2Application.KingdomRefer).
		Updates(kingdom2Application).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) DeleteKingdomFromApplication(user schema.User,
	kingdomToDeleteFromApplication DeleteKingdomFromApplication) error {

	var tx *gorm.DB = r.db

	var app schema.RulerApplication

	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", kingdomToDeleteFromApplication.ApplicationId).
		Where("creator_refer = ?", user.Id).
		First(&app).Error
	if err != nil {
		return err
	}
	if app == (schema.RulerApplication{}) {
		return errors.New("no necessary application found")
	}

	var kingdom2Application = schema.Kingdom2Application{
		ApplicationRefer: int(kingdomToDeleteFromApplication.ApplicationId),
		KingdomRefer:     int(kingdomToDeleteFromApplication.KingdomId),
	}

	err = r.db.Where("application_refer = ? AND kingdom_refer = ?",
		kingdom2Application.ApplicationRefer, kingdom2Application.KingdomRefer).
		Delete(&kingdom2Application).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) DeleteApplication(user schema.User, applicationToDelete schema.RulerApplication) error {
	var tx *gorm.DB = r.db

	err := tx.Where("id = ?", applicationToDelete.Id).Delete(&schema.RulerApplication{}).Error
	if err != nil {
		return err
	}

	return nil
}

// func (r *Repository) GetUserApplicationsWithKingdoms(user schema.User, applicationId string) (StructApplicationWithKingdoms, error) {
// 	var applicationsToReturn StructApplicationWithKingdoms
// 	var applications []schema.RullerApplication
// 	var kingdom2Application []schema.Kingdom2Application

// 	var tx *gorm.DB = r.db

// 	err := tx.Where("creator_refer = ?", user.Id).Find(&applications).Error
// 	if err != nil {
// 		return StructApplicationWithKingdoms{}, err
// 	}

// 	if len(applications) == 0 {
// 		return StructApplicationWithKingdoms{}, errors.New("application not found")
// 	}

// 	err = tx.Where("application_refer = ?", applicationId).Find(&kingdom2Application).Error
// 	if err != nil {
// 		return StructApplicationWithKingdoms{}, err
// 	}

// 	if len(kingdom2Application) == 0 {
// 		return StructApplicationWithKingdoms{}, errors.New("application not found")
// 	}

// 	for i := 0; i < len(kingdom2Application); i++ {
// 		var nestedKingdom schema.Kingdom
// 		err = tx.Where("id = ?", kingdom2Application[i].KingdomRefer).Find(&nestedKingdom).Error
// 		if err != nil {
// 			return StructApplicationWithKingdoms{}, err
// 		}

// 		var kingdomFromApplication KingdomFromApplication
// 		kingdomFromApplication.Kingdom = nestedKingdom
// 		kingdomFromApplication.From = kingdom2Application[i].From
// 		kingdomFromApplication.To = kingdom2Application[i].To

// 		applicationsToReturn.Kingdoms = append(applicationsToReturn.Kingdoms, kingdomFromApplication)
// 	}

// 	return applicationsToReturn, nil
// }

// func (r *Repository) GetRulers(requestBody requestsModels.GetRulersRequest) ([]schema.Ruler, error) {
// 	var rulersToReturn []schema.Ruler

// 	var tx *gorm.DB = r.db

// 	switch {
// 	case requestBody.Num == 0 && requestBody.State != "":
// 		err := tx.Where("state = ?", requestBody.State).Find(&rulersToReturn).Error
// 		if err != nil {
// 			return []schema.Ruler{}, err
// 		}

// 	case requestBody.Num != 0 && requestBody.State == "":
// 		err := tx.Limit(requestBody.Num).Find(&rulersToReturn).Error
// 		if err != nil {
// 			return []schema.Ruler{}, err
// 		}

// 	case requestBody.Num != 0 && requestBody.State != "":
// 		err := tx.Where("state = ?", requestBody.State).
// 			Limit(requestBody.Num).
// 			Find(&rulersToReturn).Error
// 		if err != nil {
// 			return []schema.Ruler{}, err
// 		}

// 	default:
// 		err := tx.Find(&rulersToReturn).Error
// 		if err != nil {
// 			return []schema.Ruler{}, err
// 		}
// 	}

// 	if rulersToReturn == nil {
// 		return []schema.Ruler{}, errors.New("no necessary rulers found")
// 	}

// 	return rulersToReturn, nil
// }

// func (r *Repository) GetRuler(ruler schema.Ruler) (schema.Ruler, error) {
// 	var rulerToReturn schema.Ruler

// 	err := r.db.Where(ruler).First(&rulerToReturn).Error
// 	if err != nil {
// 		return schema.Ruler{}, err
// 	} else {
// 		return rulerToReturn, nil
// 	}
// }

// func (r *Repository) CreateKingdom(kingdom schema.Kingdom) error {
// 	return r.db.Create(&kingdom).Error
// }

// func (r *Repository) EditKingdom(kingdom schema.Kingdom) error {
// 	return r.db.Model(&schema.Kingdom{}).
// 		Where("name = ?", kingdom.Name).
// 		Updates(kingdom).Error
// }

// func (r *Repository) CreateRuler(ruler schema.Ruler) error {
// 	return r.db.Create(&ruler).Error
// }

// func (r *Repository) CreateRulerForKingdom(requestBody requestsModels.CreateRulerForKingdomRequest) error {
// 	err := r.CreateRuler(requestBody.Ruler)
// 	if err != nil {
// 		return err
// 	}

// 	ruling := schema.Ruling{
// 		BeginGoverning: requestBody.BeginGoverning,
// 		Ruler:          requestBody.Ruler,
// 		Kingdom:        requestBody.Kingdom,
// 	}

// 	return r.db.Create(&ruling).Error
// }

// func (r *Repository) EditRuler(ruler schema.Ruler) error {
// 	return r.db.Model(&schema.Ruler{}).
// 		Where("name = ?", ruler.Name).
// 		Updates(ruler).Error
// }

// func (r *Repository) GetUserRole(username string) (role.Role, error) {
// 	user := &schema.User{}

// 	err := r.db.Where("name = ?", username).First(&user).Error
// 	if err != nil {
// 		return role.Unknown, err
// 	}
// 	if user == (&schema.User{}) {
// 		return role.Unknown, errors.New("no user found")
// 	}

// 	return user.Role, nil
// }

// func (r *Repository) RulerStateChange(id int, state string) error {
// 	return r.db.Model(&schema.Ruler{}).
// 		Where("id = ?", id).
// 		Update("state", state).Error
// }

// func (r *Repository) DeleteKingdom(kingdomName string) error {
// 	return r.db.Model(&schema.Kingdom{}).
// 		Where("name = ?", kingdomName).
// 		Update("state", "Захвачено ящерами").Error
// }

// func (r *Repository) DeleteRuler(rulerName string) error {
// 	return r.db.Model(&schema.Ruler{}).
// 		Where("name = ?", rulerName).
// 		Update("status", "Помер").Error
// }

// func (r *Repository) DeleteKingdomRuler(kingdomName string, rulerName string, rulingID int) error {
// 	return r.db.Where("kingdom_name = ?", kingdomName).
// 		Where("ruler_name = ?", rulerName).
// 		Where("id = ?", rulingID).Delete(&schema.Ruling{}).Error
// }

func (r *Repository) Signup(user *schema.User) error {
	userCheck, err := r.GetUserByName(user.Name)
	if err != nil {
		fmt.Println(err)
	}

	if userCheck != nil {
		return errors.New("user already existed")
	}

	if user.UUID == uuid.Nil {
		user.UUID = uuid.New()
	}

	return r.db.Create(user).Error
}

func (r *Repository) GetUserByName(name string) (*schema.User, error) {
	user := &schema.User{
		Name: name,
	}

	err := r.db.Where("name = ?", name).First(&user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}
