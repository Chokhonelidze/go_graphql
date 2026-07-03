package model

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Songs struct {
	ID         int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     uuid.UUID `gorm:"column:user_id;index" json:"user_id"`
	SongID     int       `gorm:"column:song_id;index" json:"song_id"`
	PlayCount  int       `gorm:"column:play_count" json:"play_count"`
	Title      string    `gorm:"column:title;type:varchar(500);index" json:"title"`
	Release    string    `gorm:"column:release;type:varchar(300)" json:"release"`
	ArtistName string    `gorm:"column:artist_name;type:varchar(300);index" json:"artist"`
	Year       int       `gorm:"column:year;type:integer" json:"year"`
	LocalLink  string    `gorm:"column:local_link;type:varchar(1000)" json:"local_link"`
}

func MigrateSongs(db *gorm.DB) error {
	if err := db.AutoMigrate(&Songs{}); err != nil {
		return err
	}

	var count int64
	if err := db.Model(&Songs{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // Data already seeded, stop execution safely
	}

	csvPath := filepath.Join("graph", "final_data.csv")
	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open songs csv: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	rows, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("read songs csv: %w", err)
	}

	userIDMap := make(map[int]uuid.UUID)

	// FIX 1: Pre-allocate precise slice capacity to prevent RAM spikes / out of memory errors
	totalRecords := len(rows) - 1
	if totalRecords <= 0 {
		return nil
	}
	songsToInsert := make([]Songs, 0, totalRecords)

	for _, row := range rows[1:] {
		if len(row) < 8 {
			continue
		}
		userID, _ := strconv.Atoi(row[1])
		songID, _ := strconv.Atoi(row[2])
		playCount, _ := strconv.Atoi(row[3])
		year, _ := strconv.Atoi(row[7])

		if _, exists := userIDMap[userID]; !exists {
			userIDMap[userID] = uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("user:%d", userID)))
		}

		item := Songs{
			UserID:     userIDMap[userID],
			SongID:     songID,
			PlayCount:  playCount,
			Title:      row[4],
			Release:    row[5],
			ArtistName: row[6],
			Year:       year,
		}

		// FIX 2: Instead of running db.Create here, add it to the memory block
		songsToInsert = append(songsToInsert, item)
	}

	// FIX 3: Batch update all 117k entries down to 2,000 per query. Safe and very fast.
	if len(songsToInsert) > 0 {
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.CreateInBatches(songsToInsert, 2000).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("batch insert song rows: %w", err)
		}
	}

	return nil
}
