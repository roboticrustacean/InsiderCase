package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/roboticrustacean/InsiderCase/internal/league"
	"github.com/roboticrustacean/InsiderCase/internal/storage"
	"gorm.io/gorm"
)

// Global variables to store the database connection, schedule, teams, and current week
var (
	db             *gorm.DB
	schedule       league.Schedule
	currentWeek    int
	teams          []league.Team
	teamMap        map[uint]*league.Team
	showAllResults bool
	mu             sync.Mutex // To handle concurrent access to showAllResults
)

// initDatabase initializes the database connection and fetches the teams from the database
func initDatabase() {
	var err error
	db, err = storage.SetupDatabase()
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	if db == nil {
		log.Fatal("Database connection is nil after setup")
	}

	log.Println("Database connection established")

	// Check if teams exist in the database, if not, initialize default teams (current implementation resets db on every restart, so this is currently redundant)
	if result := db.Find(&teams); result.Error != nil || len(teams) != 4 {
		log.Println("Teams not found or incorrect number of teams in database, initializing default teams")
		// Initialize default teams
		teams = []league.Team{
			{Name: "Chelsea", Strength: 5},
			{Name: "Arsenal", Strength: 4},
			{Name: "Manchester City", Strength: 3},
			{Name: "Liverpool", Strength: 2},
		}
		// Reset teams table
		db.Exec("DELETE FROM teams")
		for _, team := range teams {
			if result := db.Create(&team); result.Error != nil {
				log.Fatalf("Failed to create team %s: %v", team.Name, result.Error)
			}
		}
	} else {
		log.Println("Teams successfully fetched from database")
	}
	// Fetch teams again to get the IDs
	db.Find(&teams)
	teamMap = make(map[uint]*league.Team)
	for i := range teams {
		teamMap[teams[i].ID] = &teams[i]
	}
	// Create match schedule
	schedule = league.CreateMatchSchedule(db, teams)
	//league.PrintSchedule(schedule, teamMap)
	currentWeek = 0
}

func main() {
	storage.ResetDatabase() // Reset the database on every restart
	initDatabase()          // Initialize the database connection and fetch the teams

	r := gin.Default() // Initialize the Gin router

	r.GET("/", homeHandler)              // Define the route for the home page
	r.GET("/next_week", nextWeekHandler) // Define the route for the next week simulation
	r.GET("/play_all", playAllHandler)   // Define the route for simulating all remaining weeks

	// Start the server on port 8080
	log.Println("Server started at :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// homeHandler is the handler function for the home page
func homeHandler(c *gin.Context) {
	log.Println("homeHandler invoked")
	if db == nil {
		log.Println("Database connection is nil in homeHandler")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	// Fetch all matches from the database
	var matches []league.Match
	if result := db.Find(&matches); result.Error != nil {
		log.Printf("Failed to fetch matches: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	// Define a struct to store the match data to be displayed on the home page
	type MatchView struct {
		HomeTeamName string
		AwayTeamName string
		HomeGoals    int
		AwayGoals    int
	}
	// Create a slice to store the match data
	var matchViews []MatchView
	if showAllResults {
		// Show results of all matches (when "Play All" button is clicked)
		for _, match := range matches {
			homeTeam := teamMap[match.HomeTeamID]
			awayTeam := teamMap[match.AwayTeamID]
			matchViews = append(matchViews, MatchView{
				HomeTeamName: homeTeam.Name,
				AwayTeamName: awayTeam.Name,
				HomeGoals:    match.HomeGoals,
				AwayGoals:    match.AwayGoals,
			})
		}
	} else {
		// Show results of the current week only
		for _, match := range matches {
			if match.Week == currentWeek {
				homeTeam := teamMap[match.HomeTeamID]
				awayTeam := teamMap[match.AwayTeamID]
				matchViews = append(matchViews, MatchView{
					HomeTeamName: homeTeam.Name,
					AwayTeamName: awayTeam.Name,
					HomeGoals:    match.HomeGoals,
					AwayGoals:    match.AwayGoals,
				})
			}
		}
	}
	// Sort teams by points to display on league table
	sortedTeams := make([]league.Team, len(teams)) // Create a copy of the teams slice
	copy(sortedTeams, teams)
	// Sort the teams by points in descending order
	sort.Slice(sortedTeams, func(i, j int) bool {
		return sortedTeams[i].Points > sortedTeams[j].Points
	})
	// Calculate the championship predictions if the current week is 4 or more
	var predictions map[string]int
	if currentWeek >= 4 {
		predictions = league.CalculateChampionshipPredictions(teams)
	} else {
		predictions = make(map[string]int) // Return an empty map if the current week is less than 4
	}
	// Define a struct to store the prediction data
	type Prediction struct {
		Team       string
		Percentage int
	}
	// Create a slice to store the prediction data
	var predictionList []Prediction
	for team, percentage := range predictions {
		predictionList = append(predictionList, Prediction{
			Team:       team,
			Percentage: percentage,
		})
	}
	// Sort the predictions by percentage in descending order
	sort.Slice(predictionList, func(i, j int) bool {
		return predictionList[i].Percentage > predictionList[j].Percentage
	})
	// Load the template file
	tmplPath := filepath.Join("..", "templates", "index.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		log.Printf("Failed to parse template: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	// Define the data to be passed to the template index.html
	data := struct {
		CurrentWeek int
		Teams       []league.Team
		Matches     []MatchView
		Predictions []Prediction
	}{
		CurrentWeek: currentWeek,
		Teams:       sortedTeams,
		Matches:     matchViews,
		Predictions: predictionList,
	}

	if err := tmpl.Execute(c.Writer, data); err != nil {
		log.Printf("Failed to execute template: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
	}
}

// nextWeekHandler is the handler function for the "Next Week" button
func nextWeekHandler(c *gin.Context) {
	log.Println("nextWeekHandler invoked")
	if db == nil {
		log.Println("Database connection is nil in nextWeekHandler")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	league.SimulateNextWeek(db, schedule, &currentWeek, teamMap) // Simulate the next week of matches

	c.Redirect(http.StatusFound, "/") // Redirect to the home page
}

// playAllHandler is the handler function for the "Play All" button
func playAllHandler(c *gin.Context) {
	log.Println("playAllHandler invoked")
	if db == nil {
		log.Println("Database connection is nil in playAllHandler")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	league.SimulateAllRemainingWeeks(db, schedule, &currentWeek, teamMap) // Simulate all remaining weeks of matches

	mu.Lock() // Lock the mutex to prevent concurrent access to showAllResults
	showAllResults = true
	mu.Unlock() // Unlock the mutex

	c.Redirect(http.StatusFound, "/") // Redirect to the home page
}
