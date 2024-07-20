package league

import (
	"gorm.io/gorm"
)

// Team struct represents a single team in the league
type Team struct {
	gorm.Model

	Name           string `gorm:"not null"`
	Points         int
	Played         int
	Won            int
	Drawn          int
	Lost           int
	GoalsScored    int
	GoalsConceded  int
	GoalDifference int
	Strength       int `gorm:"not null"`
}
