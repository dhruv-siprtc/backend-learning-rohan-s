package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	Name      string `json:"name"`
	Email     string `json:"email" gorm:"uniqueIndex"`
	Password  string `json:"-"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
