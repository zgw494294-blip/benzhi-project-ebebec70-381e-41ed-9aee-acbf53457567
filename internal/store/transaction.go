package store

import "database/sql"

func withTx(db *sql.DB, fn func(*sql.Tx) error) error {
	tx, e := db.Begin()
	if e != nil {
		return e
	}
	if e = fn(tx); e != nil {
		_ = tx.Rollback()
		return e
	}
	return tx.Commit()
}
func execTx(tx *sql.Tx, q string, args ...any) error { _, e := tx.Exec(q, args...); return e }
