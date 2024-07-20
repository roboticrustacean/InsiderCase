package storage

import (
	"os"

	"github.com/roboticrustacean/InsiderCase/internal/league"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func SetupDatabase() (*gorm.DB, error) {
	dsn := os.Getenv("MYSQL_DSN")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&league.Team{}, &league.Match{})
	return db, nil
}

func ResetDatabase() error {
	dsn := os.Getenv("MYSQL_DSN")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	// Get the migrator to manage database schema
	migrator := db.Migrator()

	// Drop all tables
	tables, err := migrator.GetTables()
	if err != nil {
		return err
	}
	for _, table := range tables {
		if err := migrator.DropTable(table); err != nil {
			return err
		}
	}

	// Reapply migrations
	if err := db.AutoMigrate(&league.Team{}, &league.Match{}); err != nil {
		return err
	}

	return nil
}
