package model
type ZurichTeam struct {
	ID        uint   `gorm:"primaryKey"`
	Name      *string `gorm:"size:100;not null;uniqueIndex"`
}