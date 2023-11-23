package processing

import (
	"errors"
	requests "kingdoms/internal/database/requestModel"
	schema "kingdoms/internal/database/schema"

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

func (r *Repository) GetKingdoms(requestBody requests.GetKingdomsRequest) ([]schema.Kingdom, error) {
	kingdomsToReturn := []schema.Kingdom{}

	var tx *gorm.DB = r.db

	switch {
	case requestBody.KingdomName == "" && requestBody.RulerName == "All" && requestBody.State != "": // TODO
		err := tx.Where("state = ?", requestBody.State).Find(&kingdomsToReturn).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}

	case requestBody.KingdomName != "" && requestBody.RulerName == "All" && requestBody.State != "": // TODO
		err := tx.Where("state = ?", requestBody.State).Find(&kingdomsToReturn).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}

	case requestBody.KingdomName == "" && requestBody.RulerName != "All" && requestBody.State == "": // TODO
		var ruler schema.Ruler
		err := tx.Where("name LIKE " + "'%" + requestBody.RulerName + "%'").First(&ruler).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}
		if ruler == (schema.Ruler{}) {
			return []schema.Kingdom{}, errors.New("no necessary ruler")
		}

		err = tx.Joins("JOIN rulings ON rulings.kingdom_refer = kingdoms.id").
			Where("ruling.ruler_refer = ?", ruler.Id).
			Find(&kingdomsToReturn).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}

	case requestBody.KingdomName != "" && requestBody.RulerName != "All" && requestBody.State == "": // TODO
		var ruler schema.Ruler
		err := tx.Where("name LIKE " + "'%" + requestBody.RulerName + "%'").First(&ruler).Error

		if err != nil {
			return []schema.Kingdom{}, err
		}
		if ruler == (schema.Ruler{}) {
			return []schema.Kingdom{}, errors.New("no necessary ruler")
		}

		err = tx.Joins("JOIN rulings ON rulings.kingdom_refer = kingdoms.id").
			Where("ruling.ruler_refer = ?", ruler.Id).
			Find(&kingdomsToReturn).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}

	case requestBody.KingdomName == "" && requestBody.RulerName != "All" && requestBody.State != "": // TODO
		var ruler schema.Ruler
		err := tx.Where("name = ?", requestBody.RulerName).First(&ruler).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}
		if ruler == (schema.Ruler{}) {
			return []schema.Kingdom{}, errors.New("no necessary ruler")
		}

		err = tx.Joins("JOIN rulings ON rulings.kingdom_refer = kingdoms.id").
			Where("ruling.ruler_refer = ?", ruler.Id).
			Where("kingdoms.state = ?", requestBody.State).
			Find(&kingdomsToReturn).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}

	case requestBody.KingdomName != "" && requestBody.RulerName != "All" && requestBody.State != "": // TODO
		var ruler schema.Ruler
		err := tx.Where("name = ?", requestBody.RulerName).First(&ruler).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}
		if ruler == (schema.Ruler{}) {
			return []schema.Kingdom{}, errors.New("no necessary ruler")
		}

		err = tx.Joins("JOIN rulings ON rulings.kingdom_refer = kingdoms.id").
			Where("ruling.ruler_refer = ?", ruler.Id).
			Where("kingdoms.state = ?", requestBody.State).
			Find(&kingdomsToReturn).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}

	case requestBody.KingdomName != "" && requestBody.RulerName == "All" && requestBody.State == "": // TODO
		err := tx.Where("name LIKE " + "'%" + requestBody.KingdomName + "%'").Find(&kingdomsToReturn).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}

	default:
		err := tx.Find(&kingdomsToReturn).Error
		if err != nil {
			return []schema.Kingdom{}, err
		}
	}

	if kingdomsToReturn == nil {
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

func (r *Repository) GetRulers(requestBody requests.GetRulersRequest) ([]schema.Ruler, error) {
	var rulersToReturn []schema.Ruler

	var tx *gorm.DB = r.db

	switch {
	case requestBody.Num == 0 && requestBody.State != "":
		err := tx.Where("state = ?", requestBody.State).Find(&rulersToReturn).Error
		if err != nil {
			return []schema.Ruler{}, err
		}

	case requestBody.Num != 0 && requestBody.State == "":
		err := tx.Limit(requestBody.Num).Find(&rulersToReturn).Error
		if err != nil {
			return []schema.Ruler{}, err
		}

	case requestBody.Num != 0 && requestBody.State != "":
		err := tx.Where("state = ?", requestBody.State).
			Limit(requestBody.Num).
			Find(&rulersToReturn).Error
		if err != nil {
			return []schema.Ruler{}, err
		}

	default:
		err := tx.Find(&rulersToReturn).Error
		if err != nil {
			return []schema.Ruler{}, err
		}
	}

	if rulersToReturn == nil {
		return []schema.Ruler{}, errors.New("no necessary rulers found")
	}

	return rulersToReturn, nil
}

func (r *Repository) GetRuler(ruler schema.Ruler) (schema.Ruler, error) {
	var rulerToReturn schema.Ruler

	err := r.db.Where(ruler).First(&rulerToReturn).Error
	if err != nil {
		return schema.Ruler{}, err
	} else {
		return rulerToReturn, nil
	}
}

func (r *Repository) CreateKingdom(kingdom schema.Kingdom) error {
	return r.db.Create(&kingdom).Error
}

func (r *Repository) EditKingdom(kingdom schema.Kingdom) error {
	return r.db.Model(&schema.Kingdom{}).
		Where("name = ?", kingdom.Name).
		Updates(kingdom).Error
}

func (r *Repository) CreateRuler(ruler schema.Ruler) error {
	return r.db.Create(&ruler).Error
}

func (r *Repository) CreateRulerForKingdom(requestBody requests.CreateRulerForKingdomRequest) error {
	err := r.CreateRuler(requestBody.Ruler)
	if err != nil {
		return err
	}

	ruling := schema.Ruling{
		BeginGoverning: requestBody.BeginGoverning,
		Ruler:          requestBody.Ruler,
		Kingdom:        requestBody.Kingdom,
	}

	return r.db.Create(&ruling).Error
}

func (r *Repository) EditRuler(ruler schema.Ruler) error {
	return r.db.Model(&schema.Ruler{}).
		Where("name = ?", ruler.Name).
		Updates(ruler).Error
}

func (r *Repository) GetUserRole(username string) (string, error) {
	user := &schema.User{}

	err := r.db.Where("name = ?", username).First(&user).Error
	if err != nil {
		return "", err
	}
	if user == (&schema.User{}) {
		return "", errors.New("no user found")
	}

	return user.Rank, nil
}

func (r *Repository) RulerStateChange(id int, state string) error {
	return r.db.Model(&schema.Ruler{}).
		Where("id = ?", id).
		Update("state", state).Error
}

func (r *Repository) DeleteKingdom(kingdomName string) error {
	return r.db.Model(&schema.Kingdom{}).
		Where("name = ?", kingdomName).
		Update("state", "Захвачено ящерами").Error
}

func (r *Repository) DeleteRuler(rulerName string) error {
	return r.db.Model(&schema.Ruler{}).
		Where("name = ?", rulerName).
		Update("status", "Помер").Error
}

func (r *Repository) DeleteKingdomRuler(kingdomName string, rulerName string, rulingID int) error {
	return r.db.Where("kingdom_name = ?", kingdomName).
		Where("ruler_name = ?", rulerName).
		Where("id = ?", rulingID).Delete(&schema.Ruling{}).Error
}
