package league

import (
	"gorm.io/gorm"
)

// Match struct represents a single match between two teams
type Match struct {
	gorm.Model
	Week       int `gorm:"not null"`
	HomeTeamID uint
	AwayTeamID uint
	HomeGoals  int
	AwayGoals  int
}
