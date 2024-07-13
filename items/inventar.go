package items

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
)

type Item struct {
	Name    string `json:"name"`
	Price   int    `json:"price"`
	Image   string `json:"image"`
	Amount  int    `json:"amount"`
	Id      string `json:"id"`
	Barcode string `json:"barcode"`
}

func VerifyItemsTableExists(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS items (
    		id VARCHAR (36) PRIMARY KEY,
    		name VARCHAR (64) UNIQUE NOT NULL,
    		price INTEGER,
    		image bytea,
    		amount INTEGER,
    		barcode VARCHAR (128)
		)`)
	return err
}

func GetALlItems(db *sql.DB) ([]Item, error) {
	items := make([]Item, 0)

	result, err := db.Query(`SELECT id, name, price, image, amount, barcode FROM items`)
	defer result.Close()
	if err != nil {
		return nil, err
	}
	for result.Next() {
		var item Item
		var imageData []byte
		err = result.Scan(&item.Id, &item.Name, &item.Price, &imageData, &item.Amount, &item.Barcode)
		item.Image = base64.StdEncoding.EncodeToString(imageData)
		if err != nil {
			log.Println("Error reading results:", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func GetItemByName(name string, db *sql.DB) (Item, error) {
	result, err := db.Query("SELECT id, name, price, image, amount, barcode FROM items WHERE name = $1", name)
	defer result.Close()
	if err != nil {
		return Item{}, err
	}
	if !result.Next() {
		return Item{}, errors.New("no such item")
	}
	var item Item
	var imageData []byte
	err = result.Scan(&item.Id, &item.Name, &item.Price, &imageData, &item.Amount, &item.Barcode)
	item.Image = base64.StdEncoding.EncodeToString(imageData)
	return item, err
}

func GetItemById(id string, db *sql.DB) (Item, error) {
	result, err := db.Query("SELECT id, name, price, image, amount, barcode FROM items WHERE id = $1", id)
	defer result.Close()
	if err != nil {
		return Item{}, err
	}
	if !result.Next() {
		return Item{}, errors.New("no such item")
	}
	var item Item
	var imageData []byte
	err = result.Scan(&item.Id, &item.Name, &item.Price, &imageData, &item.Amount, &item.Barcode)
	item.Image = base64.StdEncoding.EncodeToString(imageData)
	return item, err
}

func GetItemByBarcode(barcode string, db *sql.DB) (Item, error) {
	result, err := db.Query("SELECT id, name, price, image, amount, barcode FROM items WHERE barcode = $1", barcode)
	defer result.Close()
	if err != nil {
		return Item{}, err
	}
	if !result.Next() {
		return Item{}, errors.New("no such item")
	}
	var item Item
	var imageData []byte
	err = result.Scan(&item.Id, &item.Name, &item.Price, &imageData, &item.Amount, &item.Barcode)
	item.Image = base64.StdEncoding.EncodeToString(imageData)
	return item, err
}

func InsertNewItem(item *Item, db *sql.DB) error {
	imageData, err := base64.StdEncoding.DecodeString(item.Image)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO items (id, name, price, image, amount, barcode) VALUES ($1, $2, $3, $4, $5, $6)",
		item.Id, item.Name, item.Price, imageData, item.Amount, item.Barcode)
	return err
}

func UpdateItemWithTransaction(item *Item, tx *sql.Tx) error {
	imageData, err := base64.StdEncoding.DecodeString(item.Image)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE items SET name = $1, price = $2, image = $3, amount = $4, barcode = $5 WHERE id = $6",
		item.Name, item.Price, imageData, item.Amount, item.Barcode, item.Id)
	return err
}

func UpdateItem(item *Item, tx *sql.DB) error {
	imageData, err := base64.StdEncoding.DecodeString(item.Image)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE items SET name = $1, price = $2, image = $3, amount = $4, barcode = $5 WHERE id = $6",
		item.Name, item.Price, imageData, item.Amount, item.Barcode, item.Id)
	return err
}
