package collection

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type (
	Database struct {
		db              *sql.DB
		Driver          string
		DSN             string
		MaxOpenConns    int
		MaxIdleConns    int
		ConnMaxLifetime time.Duration
	}

	Databases map[string]*Database
)

func (dbs Databases) Get(key string) *sql.DB {
	return dbs[key].db
}

func (dbs Databases) Set() (err error) {
	for _, v := range dbs {
		if v.db, err = sql.Open(v.Driver, v.DSN); err != nil {
			return err
		}

		v.db.SetMaxOpenConns(v.MaxOpenConns)
		v.db.SetMaxIdleConns(v.MaxIdleConns)
		v.db.SetConnMaxLifetime(v.ConnMaxLifetime)

		if err = v.db.Ping(); err != nil {
			return err
		}
	}

	return nil
}

func (dbs Databases) Close(label string) (err error) {
	for _, v := range dbs {
		if v.db != nil {
			err = v.db.Close()
		}
	}

	return err
}
