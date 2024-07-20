package league

import (
	"log"
	"math/rand"

	"gorm.io/gorm"
)

// Shedule struct represents a schedule of matches
type Schedule struct {
	Matches [][]Match
}

// CreateMatchSchedule generates a schedule of matches for the given teams
func CreateMatchSchedule(db *gorm.DB, teams []Team) Schedule {
	var schedule Schedule
	numTeams := len(teams)

	// Create an empty slice to hold all the matches
	var matches []Match

	// Generate matches using round-robin algorithm
	for round := 0; round < (numTeams-1)*2; round++ {
		for i := 0; i < numTeams/2; i++ {
			home := (round + i) % (numTeams - 1)
			away := (numTeams - 1 - i + round) % (numTeams - 1)
			if i == 0 {
				away = numTeams - 1
			}
			if round%2 == 0 {
				matches = append(matches, Match{HomeTeamID: teams[home].ID, AwayTeamID: teams[away].ID})
			} else {
				matches = append(matches, Match{HomeTeamID: teams[away].ID, AwayTeamID: teams[home].ID})
			} // Future proofing for when we have more than 4 teams
		}
	}

	// Shuffle matches to randomize the schedule
	rand.Shuffle(len(matches), func(i, j int) {
		matches[i], matches[j] = matches[j], matches[i]
	})

	// Distribute matches into weeks with two matches per week
	for i := 0; i < len(matches); i += 2 {
		var weekMatches []Match
		week := i / 2 // Calculate the week number
		if i < len(matches) {
			matches[i].Week = week + 1 // Set the week for the match
			db.Create(&matches[i])
			weekMatches = append(weekMatches, matches[i])

		}
		if i+1 < len(matches) {
			matches[i+1].Week = week + 1 // Set the week for the match
			db.Create(&matches[i+1])
			weekMatches = append(weekMatches, matches[i+1])

		}
		schedule.Matches = append(schedule.Matches, weekMatches)
	}

	return schedule
}

// prints the created schedule (for debugging purposes)
func PrintSchedule(schedule Schedule, teams map[uint]*Team) {
	for i, week := range schedule.Matches {
		println("Week", i+1)
		for _, match := range week {
			home := teams[match.HomeTeamID]
			away := teams[match.AwayTeamID]
			println(home.Name, "vs", away.Name)
			println("Week", match.Week)
		}
		println()
	}

}

// SimulateMatch simulates a match between two teams and returns the number of goals scored by each team
func SimulateMatch(home, away *Team) (int, int) {
	homeGoals := rand.Intn(home.Strength + 1)
	awayGoals := rand.Intn(away.Strength + 1)

	// Update goals scored and conceded
	home.GoalsScored += homeGoals
	home.GoalsConceded += awayGoals
	away.GoalsScored += awayGoals
	away.GoalsConceded += homeGoals

	// Update played matches
	home.Played++
	away.Played++

	// Update points, wins, losses, and draws
	if homeGoals > awayGoals {
		home.Won++
		away.Lost++
		home.Points += 3
	} else if awayGoals > homeGoals {
		away.Won++
		home.Lost++
		away.Points += 3
	} else {
		home.Drawn++
		away.Drawn++
		home.Points += 1
		away.Points += 1
	}

	// Update goal difference
	home.GoalDifference = home.GoalsScored - home.GoalsConceded
	away.GoalDifference = away.GoalsScored - away.GoalsConceded

	return homeGoals, awayGoals
}

// SimulateWeekMatches simulates all matches in a given week and updates the teams and matches in the database
func SimulateWeekMatches(db *gorm.DB, schedule Schedule, week int, teams map[uint]*Team) {
	// Check if the week is within the schedule range
	if week >= len(schedule.Matches) {
		return
	}

	for _, match := range schedule.Matches[week] {
		// Check if the match's week matches the current week parameter
		var existingMatch Match
		if err := db.Where("home_team_id = ? AND away_team_id = ? AND week = ?", match.HomeTeamID, match.AwayTeamID, week+1).First(&existingMatch).Error; err != nil {
			// If no match is found for this week, continue to simulate
			if err != gorm.ErrRecordNotFound {
				log.Printf("Error fetching match: %v", err)
				continue
			}
		}

		home := teams[match.HomeTeamID]
		away := teams[match.AwayTeamID]
		homeGoals, awayGoals := SimulateMatch(home, away)

		// Update match results and week
		match.HomeGoals = homeGoals
		match.AwayGoals = awayGoals
		match.Week = week + 1 // Set the week for the match, starting from 1

		// Save match results
		if existingMatch.ID == 0 {
			// If no existing match found, create a new one
			if err := db.Create(&match).Error; err != nil {
				log.Printf("Error creating match: %v", err)
			}
		} else {
			// If match exists, update it
			if err := db.Model(&existingMatch).Updates(Match{
				HomeGoals: match.HomeGoals,
				AwayGoals: match.AwayGoals,
			}).Error; err != nil {
				log.Printf("Error updating match: %v", err)
			}
		}

		// Save updated teams
		if err := db.Save(home).Error; err != nil {
			log.Printf("Error saving home team: %v", err)
		}
		if err := db.Save(away).Error; err != nil {
			log.Printf("Error saving away team: %v", err)
		}
	}
}

// SimulateNextWeek simulates the next week of matches and updates the current week parameter
func SimulateNextWeek(db *gorm.DB, schedule Schedule, currentWeek *int, teams map[uint]*Team) {
	if *currentWeek < len(schedule.Matches) {
		SimulateWeekMatches(db, schedule, *currentWeek, teams)
		*currentWeek++
	}
}

// SimulateAllRemainingWeeks simulates all remaining weeks of matches
func SimulateAllRemainingWeeks(db *gorm.DB, schedule Schedule, currentWeek *int, teams map[uint]*Team) {
	for *currentWeek < len(schedule.Matches) {
		SimulateNextWeek(db, schedule, currentWeek, teams)
	}
}

// CalculateChampionshipPredictions calculates the percentage chance of each team winning the championship
func CalculateChampionshipPredictions(teams []Team) map[string]int {
	totalPoints := 0
	predictions := make(map[string]int) // Map to store predictions

	// Calculate total points of all teams based on their current points
	for _, team := range teams {
		totalPoints += team.Points
	}

	for _, team := range teams {
		predictions[team.Name] = (team.Points * 100) / totalPoints
	}

	return predictions
}
