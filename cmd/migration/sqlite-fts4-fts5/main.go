package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func open(path string) (*sqlx.DB, error) {
	db := sqlx.MustConnect("sqlite3", path)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("can't connect to database: %s", err)
	}

	return db, nil
}

func migrateBookmarkContentsFTS4toFTS5(db *sqlx.DB) error {
	row := db.QueryRow(`SELECT sql
		FROM sqlite_master
		WHERE type = 'table' AND name = 'bookmark_content' AND sql LIKE '%USING fts4%'`)
	if row.Err() != nil {
		return fmt.Errorf("error during query to check bookmarks_content table: %s", row.Err())
	}

	var stmt string
	if err := row.Scan(&stmt); err != nil {
		return fmt.Errorf("no results from check query. Is your database already migrated?")
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	log.Info().Msg("Found bookmark_content table with fts4 format.")
	log.Info().Msg("This is a single time operation, and will only be performed once")
	log.Info().Msg("The time it will take depends on the bookmarks number")

	// Create new table
	log.Debug().Msg("Creating new table")
	tx.MustExec(`CREATE VIRTUAL TABLE bookmark_content_fts5
	USING fts5(title, content, html, docid)`)

	// Migrate all data from the fts4 to the fts5 table
	log.Debug().Msg("Migrating all data to new table")
	tx.MustExec(`INSERT INTO bookmark_content_fts5
		(title, content, html, docid)
		SELECT title, content, html, docid FROM bookmark_content`)

	// Remove fts4 table
	log.Debug().Msg("Removing old table")
	tx.MustExec(`DROP TABLE bookmark_content`)

	// Rename old table to keep the data
	log.Debug().Msg("Renaming new table to it's correct name")
	tx.MustExec(`ALTER TABLE bookmark_content_fts5 RENAME TO bookmark_content`)

	// Commit
	log.Debug().Msg("Committing")
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error commiting migration 0001: %s", err)
	}

	return nil
}

func cleanSchemaMigrations(db *sqlx.DB) error {
	row := db.QueryRow(`SELECT version, dirty FROM schema_migrations`)
	tableExists := row.Err() == nil || (row.Err() != nil && !strings.Contains(row.Err().Error(), "no such table"))
	if row.Err() != nil && tableExists {
		return fmt.Errorf("error during query to check schema_migrations table: %s", row.Err())
	}

	if !tableExists {
		log.Info().Msg("schema_migrations table not found. Skipping.")
		return nil
	}

	var version, dirty int
	if err := row.Scan(&version, &dirty); err != nil {
		return fmt.Errorf("error extracting information from schema_migrations table")
	}

	if dirty != 1 {
		log.Info().Msg("schema_migrations table is not dirty. Skipping.")
		return nil
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	log.Info().Msg("Updating schema_migrations table to reflect the migration")
	tx.MustExec(`UPDATE schema_migrations SET version = ?, dirty = 0`, version-1)

	// Commit
	log.Debug().Msg("Committing")
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error restoring schema_migrations to usable state: %s", err)
	}

	return nil
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().Msg("Command line utility to migrate a Shiori SQlite database from FTS4 to FTS5")

	databasePath := flag.String("path", "", "Path to the SQLite database file")
	flag.Parse()

	if *databasePath == "" {
		flag.Usage()
		return
	}

	log.Warn().Msg("Remember to make a backup of your database file!")
	log.Warn().Msg("Press any key when you are ready to proceed.")

	_, _ = os.Stdin.Read(make([]byte, 1))

	db, err := open(*databasePath)
	if err != nil {
		log.Error().Msgf("Error opening database: %s", err)
	}

	if err := migrateBookmarkContentsFTS4toFTS5(db); err != nil {
		log.Error().Msgf("error migrating database to fts5: %s", err)
	}

	if err := cleanSchemaMigrations(db); err != nil {
		log.Error().Msgf("error modifying schema_migrations: %s", err)
	}
}
