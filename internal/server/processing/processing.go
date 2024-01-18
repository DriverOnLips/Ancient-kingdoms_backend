package processing

import (
	"errors"
	"kingdoms/internal/database/schema"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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

func (r *Repository) GetKingdoms(kingdomName string) ([]schema.Kingdom, error) {
	kingdomsToReturn := []schema.Kingdom{}

	var tx *gorm.DB = r.db

	err := tx.Where("name LIKE " + "'%" + kingdomName + "%'").
		Order("id").Find(&kingdomsToReturn).Error
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

func (r *Repository) CreateKingdom(kingdom schema.Kingdom) error {
	err := r.db.Create(&kingdom).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) UpdateKingdom(kingdom schema.Kingdom) error {
	err := r.db.Model(&schema.Kingdom{}).
		Where("id = ?", kingdom.Id).
		Updates(kingdom).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) UpdateKingdomStatus(kingdomToUpdate KingdomToUpdate) error {
	res := r.db.Model(&schema.Kingdom{}).
		Where("id = ?", kingdomToUpdate.Id).
		Update("state", kingdomToUpdate.State)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (r *Repository) GetApplications(applicationId string) ([]schema.RulerApplication, error) {
	var applicationsToReturn []schema.RulerApplication

	var tx *gorm.DB = r.db

	if applicationId == "" {
		err := tx.Where("state != 'Удалена'").
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

func (r *Repository) GetAllApplications(params StructGetAllApplications) ([]schema.RulerApplication,
	error) {

	var applicationsToReturn []schema.RulerApplication
	var err error

	switch {
	case params.Status == "" && params.From == datatypes.Date{} && params.To == datatypes.Date{}:
		err = r.db.
			Where("state != 'Удалена'").
			Order("id").
			Preload("Creator").
			Find(&applicationsToReturn).Error
		break
	case params.Status == "" && params.From == datatypes.Date{} && params.To != datatypes.Date{}:
		err = r.db.
			Where("state != 'Удалена' AND date_send < ?", params.To).
			Order("id").
			Preload("Creator").
			Find(&applicationsToReturn).Error
		break
	case params.Status == "" && params.From != datatypes.Date{} && params.To == datatypes.Date{}:
		err = r.db.
			Where("state != 'Удалена' AND date_send > ?", params.From).
			Order("id").
			Preload("Creator").
			Find(&applicationsToReturn).Error
		break
	case params.Status == "" && params.From != datatypes.Date{} && params.To != datatypes.Date{}:
		err = r.db.
			Where("state != 'Удалена' AND date_send > ? AND date_send < ?", params.From, params.To).
			Order("id").
			Preload("Creator").
			Find(&applicationsToReturn).Error
		break
	case params.Status != "" && params.From == datatypes.Date{} && params.To == datatypes.Date{}:
		err = r.db.
			Where("state = ?", params.Status).
			Order("id").
			Preload("Creator").
			Find(&applicationsToReturn).Error
		break
	case params.Status != "" && params.From == datatypes.Date{} && params.To != datatypes.Date{}:
		err = r.db.
			Where("state = ? AND date_send < ?", params.Status, params.To).
			Order("id").
			Preload("Creator").
			Find(&applicationsToReturn).Error
		break
	case params.Status != "" && params.From != datatypes.Date{} && params.To == datatypes.Date{}:
		err = r.db.
			Where("state = ? AND date_send > ?", params.Status, params.From).
			Order("id").
			Preload("Creator").
			Find(&applicationsToReturn).Error
		break
	default:
		err = r.db.
			Where("state = ? AND date_send > ? AND date_send < ?",
				params.Status, params.From, params.To).
			Order("id").
			Preload("Creator").
			Find(&applicationsToReturn).Error
		break
	}

	if err != nil {
		return []schema.RulerApplication{}, err
	}

	if len(applicationsToReturn) == 0 {
		return []schema.RulerApplication{}, errors.New("no necessary ruler applications found")
	}

	return applicationsToReturn, nil
}

func (r *Repository) GetApplicationWithKingdoms(applicationId string) (StructApplicationWithKingdoms, error) {
	nestedApplication, err := r.GetApplications(applicationId)
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

func (r *Repository) CreateApplication() (schema.RulerApplication, error) {
	application := schema.RulerApplication{
		Creator:    schema.User{},
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

func (r *Repository) UpdateApplicationStatus(applicationToUpdate ApplicationToUpdate) error {
	var tx *gorm.DB = r.db

	var app schema.RulerApplication
	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", applicationToUpdate.Id).
		First(&app).Error
	if err != nil {
		return err
	}
	if app == (schema.RulerApplication{}) {
		return errors.New("no necessary application found")
	}

	switch applicationToUpdate.State {
	case "На рассмотрении":
		err = tx.Model(&schema.RulerApplication{}).
			Where("id = ?", applicationToUpdate.Id).
			Updates(map[string]interface{}{
				"state":     applicationToUpdate.State,
				"date_send": datatypes.Date(time.Now()),
			}).Error
		if err != nil {
			return err
		}

		break

	case "Одобрена":
		err = tx.Model(&schema.RulerApplication{}).
			Where("id = ?", applicationToUpdate.Id).
			Updates(map[string]interface{}{
				"state":         applicationToUpdate.State,
				"date_complete": datatypes.Date(time.Now()),
			}).Error
		if err != nil {
			return err
		}
		break

	case "Отклонена":
		err = tx.Model(&schema.RulerApplication{}).
			Where("id = ?", applicationToUpdate.Id).
			Updates(map[string]interface{}{
				"state":         applicationToUpdate.State,
				"date_complete": datatypes.Date(time.Now()),
			}).Error
		if err != nil {
			return err
		}
		break

	case "Удалена":
		err = tx.Model(&schema.RulerApplication{}).
			Where("id = ?", applicationToUpdate.Id).
			Update("state", applicationToUpdate.State).Error
		if err != nil {
			return err
		}
		break

	default:
		break
	}

	return nil
}

func (r *Repository) UpdateApplication(applicationToUpdate schema.RulerApplication) error {

	var tx *gorm.DB = r.db

	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", applicationToUpdate.Id).
		Update("ruler", applicationToUpdate.Ruler).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) AddKingdomToApplication(kingdomAddToApplication KingdomAddToApplication) error {
	var tx *gorm.DB = r.db

	var app schema.RulerApplication

	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", kingdomAddToApplication.ApplicationId).
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

func (r *Repository) UpdateKingdomFromApplication(kingdomAddToApplication KingdomAddToApplication) error {
	var tx *gorm.DB = r.db

	var app schema.RulerApplication

	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", kingdomAddToApplication.ApplicationId).
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

func (r *Repository) DeleteKingdomFromApplication(
	kingdomToDeleteFromApplication DeleteKingdomFromApplication) error {

	var tx *gorm.DB = r.db

	var app schema.RulerApplication

	err := tx.Model(&schema.RulerApplication{}).
		Where("id = ?", kingdomToDeleteFromApplication.ApplicationId).
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

func (r *Repository) DeleteApplication(applicationToDelete schema.RulerApplication) error {
	var tx *gorm.DB = r.db

	err := tx.Where("id = ?", applicationToDelete.Id).Delete(&schema.RulerApplication{}).Error
	if err != nil {
		return err
	}

	return nil
}
