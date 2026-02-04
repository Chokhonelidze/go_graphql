package model

import (
	"github.com/google/uuid"
	"time"
)

type Document struct {
	DocumentID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	GroupID          string  `gorm:"not null"`
	ClmNbr           *string `gorm:"size:100"`
	CaseNumber       *string `gorm:"size:100"`
	OriginalFilename *string `gorm:"size:255"`
	FileLocation     *string `gorm:"size:255"`
	CurrentStage     *string `gorm:"size:100"`
	CurrentStatus    *string `gorm:"size:100"`
	PercentComplete  *string `gorm:"size:50"`
	PageCount        *int
	HasMedical       *bool
	HasDemand        *bool
	HasUserFeedback  *bool
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
	UploadUser       *string `gorm:"size:255"`
	UploadUserID     *string `gorm:"size:100"`
}

