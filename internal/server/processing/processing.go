package processing

import (
	"errors"
	"kingdoms/internal/database/schema"

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

	err := tx.Where("name LIKE "+"'%"+kingdomName+"%'").
		Where("state != ?", "Данные утеряны").
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

func (r *Repository) UpdateKingdomStatus(kingdomId int, state string) error {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Exec(`UPDATE public.kingdoms SET state = ? WHERE id = ?`, state, kingdomId).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
