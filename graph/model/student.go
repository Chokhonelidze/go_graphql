package model
type Student struct {
	ID        uint  `gorm:"primaryKey`
	FirstName *string `gorm:"size:100;not null"`
	LastName  *string `gorm:"size:100;not null"`
	Number    *string `gorm:"size:100;uniqueIndex"`
}

