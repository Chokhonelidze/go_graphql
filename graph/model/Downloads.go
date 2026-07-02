package model

type Downloads struct {
	ID   int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	File string `gorm:"column:file;type:varchar(1000)" json:"file"`
	Text string `gorm:"column:text;type:text" json:"text"`
}
