package model

import (
	"log"

	"gorm.io/gorm"
)

type Song struct {
	SongID     int    `gorm:"column:song_id;primaryKey;autoIncrement" json:"song_id"`
	Title      string `gorm:"column:title;type:varchar(500);index" json:"title"`
	Release    string `gorm:"column:release;type:varchar(500)" json:"release"`
	ArtistName string `gorm:"column:artist_name;type:varchar(500);index" json:"artist_name"`
	Year       int    `gorm:"column:year;type:integer" json:"year"`
	VideoLink  string `gorm:"column:video_link;type:varchar(500)" json:"video_link"`
	LocalLink  string `gorm:"column:local_link;type:varchar(500)" json:"local_link"`
}

func MigrateSong(db *gorm.DB) error {
	// 1. Ensure the singular Song table structure exists
	if err := db.AutoMigrate(&Song{}); err != nil {
		return err
	}

	// 2. Optimization check: Skip if target catalog table is already populated
	var count int64
	if err := db.Model(&Song{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		log.Println("Song catalog table is already populated. Skipping data extraction.")
		return nil
	}

	// 3. Extract uniquely grouped song configurations from your raw data table (Songs)
	// PostgreSQL / MySQL require columns to be explicitly listed or grouped to stay deterministic
	var distinctSongs []Song
	err := db.Table("songs"). // Target your original RAW csv ingestion table name
					Select("DISTINCT ON (song_id) song_id, title, release, artist_name , year, local_link").
					Order("song_id").
					Find(&distinctSongs).Error

	if err != nil {
		return err
	}

	log.Printf("Found %d distinct song profiles to extract.", len(distinctSongs))

	// 4. Batch insert all unique catalog records into your clean 'Song' table instantly
	if len(distinctSongs) > 0 {
		err = db.Transaction(func(tx *gorm.DB) error {
			// Creates batches of 2000 records at a time safely
			if err := tx.Model(&Song{}).CreateInBatches(distinctSongs, 2000).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	log.Println("Successfully migrated distinct song catalog data!")
	return nil
}
