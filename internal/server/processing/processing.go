package processing

import (
	"errors"
	"fmt"
	"kingdoms/internal/database/schema"

	"github.com/google/uuid"
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

	err := tx.Where("name LIKE " + "'%" + kingdomName + "%'").Find(&kingdomsToReturn).Error
	if err != nil {
		return []schema.Kingdom{}, err
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
