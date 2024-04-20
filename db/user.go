package db

import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"log"
)

type (
	User struct {
		Id       int64  `json:"id"`
		Login    string `json:"login"`
		PassHash string `json:"pass_hash"`
	}
	UserWeb struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
)

func CreateUsersTable(ctx context.Context, db *sql.DB) error {
	const (
		usersTable = `
			CREATE TABLE "users" (
			"id"	INTEGER NOT NULL,
			"login"	TEXT NOT NULL,
			"pass_hash"	TEXT NOT NULL,
			PRIMARY KEY("id" AUTOINCREMENT)
		);`
	)

	if _, err := db.ExecContext(ctx, usersTable); err != nil {
		return err
	}

	return nil
}

func GenHash(s string) (string, error) {
	saltedBytes := []byte(s)
	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	hash := string(hashedBytes[:])
	return hash, nil
}

func InsertUser(ctx context.Context, db *sql.DB, login string, password string) (int64, error) {
	var q = `
	INSERT INTO users (login, pass_hash) values ($1, $2)
	`
	hash, _ := GenHash(password)
	result, err := db.ExecContext(ctx, q, login, hash)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func SelectUser(ctx context.Context, db *sql.DB, login string) (User, error) {
	var (
		user User
		err  error
	)

	var q = "SELECT id, login, pass_hash FROM users WHERE login=$1"
	err = db.QueryRowContext(ctx, q, login).Scan(&user.Id, &user.Login, &user.PassHash)
	return user, err
}

func CompareHashes(hash string, s string) error {
	incoming := []byte(s)
	existing := []byte(hash)
	return bcrypt.CompareHashAndPassword(existing, incoming)
}

func (u User) ComparePassword(password string) error {
	err := CompareHashes(u.PassHash, password)
	if err != nil {
		log.Println("auth fail")
		return err
	}

	log.Println("auth success")
	return nil
}
