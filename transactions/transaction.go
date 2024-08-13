package transactions

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Port39/go-drink/items"
	"github.com/Port39/go-drink/users"
	"github.com/google/uuid"
	"log"
	"time"
)

type Transaction struct {
	Id          string `json:"id"`
	ItemId      string `json:"itemId"`
	UserId      string `json:"userId"`
	Amount      int    `json:"amount"`
	AuthBackend string `json:"authBackend"`
	Timestamp   int64  `json:"timestamp"`
}

func VerifyTransactionTableExists(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS transactions (
    		id VARCHAR (36) PRIMARY KEY,
    		itemId VARCHAR (36),
    		userId VARCHAR (36),
    		amount INTEGER,
    		authBackend VARCHAR (16),
    		timestamp INTEGER
		)`)
	return err
}

func GetTransactionsSince(ctx context.Context, since, until int64, db *sql.DB) ([]Transaction, error) {
	transactions := make([]Transaction, 0)
	result, err := db.QueryContext(ctx, "SELECT id, itemid, userid, amount, authbackend, timestamp FROM transactions WHERE timestamp > $1 AND timestamp < $2", since, until)
	if err != nil {
		return transactions, err
	}
	defer result.Close()
	for result.Next() {
		var tr Transaction
		err = result.Scan(&tr.Id, &tr.ItemId, &tr.UserId, &tr.Amount, &tr.AuthBackend, &tr.Timestamp)
		if err != nil {
			log.Println("Error reading results:", err)
		}
		transactions = append(transactions, tr)
	}
	return transactions, nil
}

func MakeTransaction(ctx context.Context, user *users.User, item *items.Item, amount int, authBackend string, db *sql.DB) error {
	finalPrice := item.Price * amount
	if user.Credit < finalPrice {
		return errors.New("not enough credits")
	}
	if item.Amount < amount {
		return errors.New("not enough items in stock")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if !user.IsCashUser() {
		user.Credit = user.Credit - finalPrice
		err = users.UpdateUserWithTransaction(ctx, user, tx)
		if err != nil {
			err := tx.Rollback()
			if err != nil {
				return err
			}
			return err
		}
	}
	item.Amount = item.Amount - amount
	err = items.UpdateItemWithTransaction(ctx, item, tx)
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return err
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO transactions (id, itemId, userId, amount, authBackend, timestamp) 
		VALUES ($1, $2, $3, $4, $5, $6)`, uuid.New().String(), item.Id, user.Id, amount, authBackend, time.Now().Unix())
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return err
		}
		return err
	}
	return tx.Commit()
}
