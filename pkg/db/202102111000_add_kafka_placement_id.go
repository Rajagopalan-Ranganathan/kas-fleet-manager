package db

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

func addKafkaPlacementId() *gormigrate.Migration {
	type KafkaRequest struct {
		PlacementId string
	}
	return &gormigrate.Migration{
		ID: "202102111000",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&KafkaRequest{}).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Table("kafka_requests").DropColumn("placement_id").Error; err != nil {
				return err
			}
			return nil
		},
	}
}
