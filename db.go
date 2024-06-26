package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

var db *sql.DB

// openDatabase returns the database handle. It checks that the database
// schema exists and create it if it doesn't.
func openDatabase(dbPath string) (*sql.DB, error) {
	var err error
	if db, err = sql.Open("sqlite", dbPath); err != nil {
		return nil, err
	}

	if _, err = db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("Error setting WAL mode. %v", err)
		return nil, err
	}
	// Check to see if the schema already exists and create it if it isn't.
	if _, err = db.ExecContext(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS links (
			keyword TEXT PRIMARY KEY, 
			URL TEXT NOT NULL, 
			owner TEXT NOT NULL, 
			created DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	); err != nil {
		return nil, err
	}

	if _, err = db.ExecContext(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS count (
			keyword TEXT PRIMARY KEY, 
			count INTEGER DEFAULT 0
		)`,
	); err != nil {
		return nil, err
	}

	// Creating schema for sessions.
	if _, err = db.ExecContext(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS sessions (
		 	token TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			expiry REAL NOT NULL)`,
	); err != nil {
		return nil, err
	}

	if _, err = db.ExecContext(
		context.Background(),
		`CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry)`,
	); err != nil {
		return nil, err
	}

	return db, nil
}

// getUrlByKeyword takes in a keyword and returns the redirect URL.
func getUrlByKeyword(kw string) (string, error) {

	row := db.QueryRowContext(
		context.Background(),
		"SELECT URL from links where keyword=?",
		kw,
	)
	var path string
	if err := row.Scan(&path); err != nil {
		return "", err
	}

	return path, nil
}

// insertOrUpdateLink uses SQLite's upsert to insert a record if it is a
// new keyword or update an existing record if it already exists.
func insertOrUpdateLink(kw, url, owner string) error {

	if len(kw) == 0 || len(url) == 0 || len(owner) == 0 {
		return fmt.Errorf("Keyword, URL and Owners can not be blank!")
	}
	sql := `INSERT INTO links(keyword, URL, owner) VALUES (?, ?, ?)
on CONFLICT (keyword) DO
UPDATE
SET
    URL = excluded.URL,
	owner = excluded.owner
WHERE owner = ?
`

	if _, err := db.Exec(sql, kw, url, owner, owner); err != nil {
		return fmt.Errorf("error while inserting. %v", err)
	}

	return nil
}

func deleteLink(kw, user string) error {

	if _, err := db.Exec("DELETE FROM links WHERE keyword = ? AND owner = ?", kw, user); err != nil {
		return fmt.Errorf("unable to delete row. %v", err)
	}

	return nil
}

func allLinks() ([]link, error) {

	var links []link

	rows, err := db.QueryContext(
		context.Background(),
		"SELECT links.keyword, links.URL, links.owner, COALESCE(count.count, 0) as views FROM links LEFT JOIN count on links.keyword = count.keyword ORDER BY created desc")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {

		var l link
		if err := rows.Scan(&l.Keyword, &l.Url, &l.Owner, &l.Views); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

func updateCounts(i map[string]int) error {

	var tx *sql.Tx
	var err error

	tx, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	sql := `INSERT INTO count(keyword, count) VALUES (?,?)
on CONFLICT (keyword) DO
UPDATE
SET
	count = count + ?
WHERE 
    keyword = ?
`

	for k, v := range i {
		if _, err := tx.Exec(sql, k, v, v, k); err != nil {
			tx.Rollback()
			return fmt.Errorf("error updating count for %v. %v", k, err)
		}
	}

	return tx.Commit()
}
