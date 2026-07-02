package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Email     string    `gorm:"type:varchar(200);not null" json:"email"`
	Password  string    `gorm:"type:varchar(300);not null" json:"password"`
	Role      string    `gorm:"type:varchar(200);not null" json:"role"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
