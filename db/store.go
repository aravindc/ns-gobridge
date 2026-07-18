package db

import (
	"context"
	"database/sql"
	"fmt"
	"ns-gobridge/model"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// DbClient connects to Postgres. pg_sslmode is "disable" to connect without
// TLS (e.g. a local/compose Postgres) or anything else (e.g. "require") to
// connect with TLS, as needed by managed providers such as Neon.
func DbClient(postgres_host string, postgres_port string, pg_user string, pg_pass string, pg_db string, pg_sslmode string) *sql.DB {
	pgconn := pgdriver.NewConnector(
		pgdriver.WithNetwork("tcp"),
		pgdriver.WithAddr(fmt.Sprintf("%s:%s", postgres_host, postgres_port)),
		pgdriver.WithUser(pg_user),
		pgdriver.WithPassword(pg_pass),
		pgdriver.WithDatabase(pg_db),
		pgdriver.WithInsecure(pg_sslmode == "disable"),
		pgdriver.WithTimeout(5*time.Second),
		pgdriver.WithDialTimeout(5*time.Second),
		pgdriver.WithReadTimeout(5*time.Second),
		pgdriver.WithWriteTimeout(5*time.Second),
	)
	return sql.OpenDB(pgconn)
}

func SelectEntries(db_client *sql.DB) []model.Nightscoutdb {
	db := bun.NewDB(db_client, pgdialect.New())
	ctx := context.Background()
	var nsentries []model.Nightscoutdb
	err := db.NewSelect().Table("nightscoutdb").Model(&nsentries).Limit(5).Scan(ctx)
	if err != nil {
		log.Fatal("Select error: ", err)
	}
	return nsentries
}

func EntriesExist(db_client *sql.DB, ns_time int64) bool {
	db := bun.NewDB(db_client, pgdialect.New())
	ctx := context.Background()
	exists, err := db.NewSelect().Table("nightscoutdb").Where("ns_time = ?", ns_time).Exists(ctx)
	if err != nil {
		log.Fatal("Exists error: ", err)
	}
	return exists
}

func SelectLatestEntry(db_client *sql.DB) (model.Nightscoutdb, error) {
	db := bun.NewDB(db_client, pgdialect.New())
	ctx := context.Background()
	var nsentry model.Nightscoutdb
	err := db.NewSelect().Model(&nsentry).OrderExpr("ns_time DESC").Limit(1).Scan(ctx)
	return nsentry, err
}

func SelectEntriesBetween(db_client *sql.DB, from int64, to int64) ([]model.Nightscoutdb, error) {
	db := bun.NewDB(db_client, pgdialect.New())
	ctx := context.Background()
	var nsentries []model.Nightscoutdb
	err := db.NewSelect().
		Model(&nsentries).
		Where("ns_time >= ?", from).
		Where("ns_time <= ?", to).
		OrderExpr("ns_time ASC").
		Scan(ctx)
	return nsentries, err
}

func InsertEntries(db_client *sql.DB, nsItem model.Nightscoutdb) {
	db := bun.NewDB(db_client, pgdialect.New())
	ctx := context.Background()
	newNsItem := &model.Nightscoutdb{
		Sgv:         nsItem.Sgv,
		Ns_time:     nsItem.Ns_time,
		Ns_datetime: nsItem.Ns_datetime,
		Trend:       nsItem.Trend,
		Utcoffset:   nsItem.Utcoffset,
		Systime:     nsItem.Systime,
	}
	log.Info("Trying to insert: ", nsItem.Sgv, nsItem.Ns_time, nsItem.Ns_datetime, nsItem.Trend, nsItem.Utcoffset, nsItem.Systime)
	res, err := db.NewInsert().Model(newNsItem).Exec(ctx)
	log.Info("Insert result: ", res)
	if err != nil {
		log.Fatal("Insert error: ", err)
	}
}
