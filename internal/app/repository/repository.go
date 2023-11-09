package repository

import (
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"kingdoms/internal/app/database"
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

func (r *Repository) GetKingdoms(requestBody database.GetKingdomsRequest) ([]database.Kingdom, error) {
	kingdomsToReturn := []database.Kingdom{}

	var tx *gorm.DB = r.db

	switch {
	case requestBody.Ruler == "All" && requestBody.State != "":
		err := tx.Where("state = ?", requestBody.State).Find(&kingdomsToReturn).Error
		if err != nil {
			return []database.Kingdom{}, err
		}

	case requestBody.Ruler != "All" && requestBody.State == "":
		var ruler database.Ruler
		err := tx.Where("name = ?", requestBody.Ruler).First(&ruler).Error
		if err != nil {
			return []database.Kingdom{}, err
		}
		if ruler == (database.Ruler{}) {
			return []database.Kingdom{}, errors.New("no necessary ruler")
		}

		err = tx.Joins("JOIN rulings ON rulings.kingdom_refer = kingdoms.id").
			Where("ruling.ruler_refer = ?", ruler.Id).
			Find(&kingdomsToReturn).Error
		if err != nil {
			return []database.Kingdom{}, err
		}

	case requestBody.Ruler != "All" && requestBody.State != "":
		var ruler database.Ruler
		err := tx.Where("name = ?", requestBody.Ruler).First(&ruler).Error
		if err != nil {
			return []database.Kingdom{}, err
		}
		if ruler == (database.Ruler{}) {
			return []database.Kingdom{}, errors.New("no necessary ruler")
		}

		err = tx.Joins("JOIN rulings ON rulings.kingdom_refer = kingdoms.id").
			Where("ruling.ruler_refer = ?", ruler.Id).
			Where("kingdoms.state = ?", requestBody.State).
			Find(&kingdomsToReturn).Error
		if err != nil {
			return []database.Kingdom{}, err
		}

	default:
		err := tx.Find(&kingdomsToReturn).Error
		if err != nil {
			return []database.Kingdom{}, err
		}
	}

	if kingdomsToReturn == nil {
		return []database.Kingdom{}, errors.New("no necessary kingdoms found")
	}

	return kingdomsToReturn, nil
}

func (r *Repository) GetKingdom(kingdom database.Kingdom) (database.Kingdom, error) {
	var kingdomToReturn database.Kingdom

	err := r.db.Where(kingdom).First(&kingdomToReturn).Error
	if err != nil {
		return database.Kingdom{}, err
	} else {
		return kingdomToReturn, nil
	}
}

func (r *Repository) GetRulers(requestBody database.GetRulersRequest) ([]database.Ruler, error) {
	var rulersToReturn []database.Ruler

	var tx *gorm.DB = r.db

	switch {
	case requestBody.Num == 0 && requestBody.State != "":
		err := tx.Where("state = ?", requestBody.State).Find(&rulersToReturn).Error
		if err != nil {
			return []database.Ruler{}, err
		}

	case requestBody.Num != 0 && requestBody.State == "":
		err := tx.Limit(requestBody.Num).Find(&rulersToReturn).Error
		if err != nil {
			return []database.Ruler{}, err
		}

	case requestBody.Num != 0 && requestBody.State != "":
		err := tx.Where("state = ?", requestBody.State).
			Limit(requestBody.Num).
			Find(&rulersToReturn).Error
		if err != nil {
			return []database.Ruler{}, err
		}

	default:
		err := tx.Find(&rulersToReturn).Error
		if err != nil {
			return []database.Ruler{}, err
		}
	}

	if rulersToReturn == nil {
		return []database.Ruler{}, errors.New("no necessary rulers found")
	}

	return rulersToReturn, nil
}

func (r *Repository) GetRuler(ruler database.Ruler) (database.Ruler, error) {
	var rulerToReturn database.Ruler

	err := r.db.Where(ruler).First(&rulerToReturn).Error
	if err != nil {
		return database.Ruler{}, err
	} else {
		return rulerToReturn, nil
	}
}

func (r *Repository) CreateKingdom(kingdom database.Kingdom) error {
	return r.db.Create(&kingdom).Error
}

func (r *Repository) EditKingdom(kingdom database.Kingdom) error {
	return r.db.Model(&database.Kingdom{}).
		Where("name = ?", kingdom.Name).
		Updates(kingdom).Error
}

func (r *Repository) CreateRuler(ruler database.Ruler) error {
	return r.db.Create(&ruler).Error
}

func (r *Repository) CreateRulerForKingdom(requestBody database.CreateRulerForKingdomRequest) error {
	err := r.CreateRuler(requestBody.Ruler)
	if err != nil {
		return err
	}

	ruling := database.Ruling{
		BeginGoverning: requestBody.BeginGoverning,
		Ruler:          requestBody.Ruler,
		Kingdom:        requestBody.Kingdom,
	}

	return r.db.Create(&ruling).Error
}

func (r *Repository) EditRuler(ruler database.Ruler) error {
	return r.db.Model(&database.Ruler{}).
		Where("name = ?", ruler.Name).
		Updates(ruler).Error
}

func (r *Repository) GetUserRole(username string) (string, error) {
	user := &database.User{}

	err := r.db.Where("name = ?", username).First(&user).Error
	if err != nil {
		return "", err
	}
	if user == (&database.User{}) {
		return "", errors.New("no user found")
	}

	return user.Rank, nil
}

func (r *Repository) RulerStateChange(id int, state string) error {
	return r.db.Model(&database.Ruler{}).
		Where("id = ?", id).
		Update("state", state).Error
}

func (r *Repository) DeleteKingdom(kingdomName string) error {
	return r.db.Model(&database.Kingdom{}).
		Where("name = ?", kingdomName).
		Update("state", "Захвачено ящерами").Error
}

func (r *Repository) DeleteRuler(rulerName string) error {
	return r.db.Model(&database.Ruler{}).
		Where("name = ?", rulerName).
		Update("status", "Помер").Error
}

func (r *Repository) DeleteKingdomRuler(kingdomName string, rulerName string, rulingID int) error {
	return r.db.Where("kingdom_name = ?", kingdomName).
		Where("ruler_name = ?", rulerName).
		Where("id = ?", rulingID).Delete(&database.Ruling{}).Error
}