package opsutil

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// 2006-01-02 13:00:00
func DatetimeNow() string {
	return time.Now().Format(LAYOUT_DATETIME_STRING)
}

// 2006-01-02
func DateNow() string {
	return time.Now().Format(LAYOUT_DATE)
}

func DatetimeLayoutNow(layout string) string {
	return time.Now().Format(layout)
}

func ReplaceSQL(old, searchPattern string) string {
	tmpCount := strings.Count(old, searchPattern)
	for m := 1; m <= tmpCount; m++ {
		old = strings.Replace(old, searchPattern, "$"+strconv.Itoa(m), 1)
	}
	return old
}

func QueryFill(query string) (new string) {
	query = strings.ReplaceAll(query, " ", "")
	split := strings.Split(query, ",")
	for range split {
		new += "?,"
	}

	return strings.TrimSuffix(new, ",")
}

func DBTransaction(db *sql.DB, txFunc func(*sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Rollback Panic
		} else if err != nil {
			tx.Rollback() // err is not nill
		} else {
			err = tx.Commit() // err is nil
		}
	}()
	err = txFunc(tx)
	return err
}

func DBTransactionPostgresMongo(dbMongo *mongo.Database, db *sql.DB, txFunc func(*sql.Tx, mongo.Session) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	mongoSession, err := dbMongo.Client().StartSession()
	if err != nil {
		return err
	}

	ctx := context.TODO()
	defer mongoSession.EndSession(ctx)

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			mongoSession.AbortTransaction(ctx)
			panic(p) // Rollback Panic
		} else if err != nil {
			tx.Rollback() // err is not nill
			mongoSession.AbortTransaction(ctx)
		} else {
			err = mongoSession.CommitTransaction(ctx)
			if err != nil {
				fmt.Println("Error Commit Transaction Mongo", err)
			}
			err = tx.Commit() // err is nil
			if err != nil {
				fmt.Println("Error Commit Transaction Postgres", err)
			}
		}
	}()

	// Start MongoDB transaction
	err = mongoSession.StartTransaction()
	if err != nil {
		tx.Rollback()
		return err
	}

	err = txFunc(tx, mongo.NewSessionContext(ctx, mongoSession))
	return err
}
