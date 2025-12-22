package database

import (
	"database/sql"
	"errors"
	"io/fs"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Initialize(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	if err = RunMigrations(); err != nil {
		return err
	}

	log.Println("Database initialized successfully")
	return nil
}

func RunMigrations() error {
	// Create source driver from embedded migrations
	sourceDriver, err := iofs.New(Migrations, "migrations")
	if err != nil {
		return err
	}

	// Create database driver
	dbDriver, err := sqlite3.WithInstance(DB, &sqlite3.Config{})
	if err != nil {
		return err
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return err
	}

	// Run migrations
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	version, dirty, _ := m.Version()
	log.Printf("Database migrations complete (version: %d, dirty: %v)", version, dirty)

	return nil
}

func MigrateDown() error {
	// Create source driver from embedded migrations
	sourceDriver, err := iofs.New(Migrations, "migrations")
	if err != nil {
		return err
	}

	// Create database driver
	dbDriver, err := sqlite3.WithInstance(DB, &sqlite3.Config{})
	if err != nil {
		return err
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return err
	}

	// Rollback one migration
	return m.Steps(-1)
}

func GetMigrationVersion() (uint, bool, error) {
	// Helper to check current migration state
	var migrationsFS fs.FS = Migrations
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return 0, false, err
	}

	dbDriver, err := sqlite3.WithInstance(DB, &sqlite3.Config{})
	if err != nil {
		return 0, false, err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return 0, false, err
	}

	return m.Version()
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
