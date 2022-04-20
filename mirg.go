package mirg

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type migrationFunc func(pgx.Tx) error

type migration struct {
	up, down migrationFunc
}

var migrations = map[int]migration{}

func AddMigration(key int, up, down migrationFunc) {
	migrations[key] = migration{up, down}
}

func sortedMigrationsKeys() []int {
	keys := make([]int, 0)

	for k := range migrations {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	return keys
}

type Conn struct {
	dbConn *pgxpool.Pool
}

func New(dbConn *pgxpool.Pool) *Conn {
	return &Conn{dbConn}
}

const schemaVersionTable = "schema_version"

func (c *Conn) Up() error {
	// check if there is the schema version table
	// if so then get current version and run only migrations
	// that are above the version.
	q := `
	select exists (
		select from
			information_schema.tables
		where
			table_schema LIKE 'public' and table_name = 'schema_version'
	)`

	ctx := context.Background()

	var existsSchemaTable bool
	if err := c.dbConn.QueryRow(ctx, q).Scan(&existsSchemaTable); err != nil {
		return fmt.Errorf("can't get schema version: %w", err)
	}

	// if there is no schema version table then create it and
	// set the version to 0 and run all migrations.
	if !existsSchemaTable {
		if _, err := c.dbConn.Exec(ctx, `create table schema_version (version int not null);`); err != nil {
			return fmt.Errorf("can't create schema version table: %w", err)
		}

		if _, err := c.dbConn.Exec(ctx, `insert into schema_version (version) values (0);`); err != nil {
			return fmt.Errorf("can't add zero schema version: %w", err)
		}
	}

	// get current schema version and run all migrations above this version
	getVersionQuery := `select version from schema_version limit 1`

	var schemaVersion int
	if err := c.dbConn.QueryRow(ctx, getVersionQuery).Scan(&schemaVersion); err != nil {
		return fmt.Errorf("can't get schema version: %w", err)
	}

	fmt.Printf("schema version %d\n", schemaVersion)

	newVersion := schemaVersion

	tx, _ := c.dbConn.Begin(ctx)

	for _, k := range sortedMigrationsKeys() {
		if k > schemaVersion {
			if err := migrations[k].up(tx); err != nil {
				fmt.Printf("can't run migration: %v\n", err)
				_ = tx.Rollback(ctx)
				return err
			}

			newVersion = k
		}
	}

	// update version of schema
	if _, err := tx.Exec(ctx, "update schema_version set version = $1", newVersion); err != nil {
		fmt.Printf("can't update schema version: %v\n", err)
		_ = tx.Rollback(ctx)
		return err
	}

	_ = tx.Commit(ctx)

	return nil
}

// TODO: down, to
