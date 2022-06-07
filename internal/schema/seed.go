package schema

import (
	"database/sql"
	"fmt"
)

func Seed(db *sql.DB) error {
	seeds := []string{}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, seed := range seeds {
		_, err = tx.Exec(seed)
		if err != nil {
			tx.Rollback()
			fmt.Println("error execute seed")
			return err
		}
	}

	return tx.Commit()
}
