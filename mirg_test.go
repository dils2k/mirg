package mirg

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func TestUp(t *testing.T) {
	AddMigration(1, up, down)
	AddMigration(2, up2, nil)

	dbConn, err := pgxpool.Connect(context.Background(), "postgresql://127.0.0.1:5432/acq?user=postgres&password=postgres&sslmode=disable")
	if err != nil {
		t.Fatalf("can't connect to db %v", err)
	}

	c := New(dbConn)

	if err := c.Up(); err != nil {
		t.Fatalf("can't migrate db %v", err)
	}
}

func up2(tx pgx.Tx) error {
	_, err := tx.Exec(context.Background(), "create table a (name varchar(30))")
	return err
}

func up(tx pgx.Tx) error {
	_, err := tx.Exec(context.Background(), `
	CREATE TABLE merchants (
			external_id BIGINT NOT NULL,
			secret_key VARCHAR NOT NULL,
			token VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT now(),

			PRIMARY KEY (external_id),
			UNIQUE (name),
			UNIQUE (token)
	);


	CREATE TABLE users (
			external_id BIGINT PRIMARY KEY NOT NULL,
			merchant_id BIGINT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT now(),

			FOREIGN KEY(merchant_id) REFERENCES merchants(external_id)
	);`)

	return err
}

func down(tx pgx.Tx) error {
	_, err := tx.Exec(context.Background(), `
	drop table merchants;
	drop table users;
	`)

	return err
}
