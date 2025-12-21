package migrations

import (
	"context"
	"embed"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/migrations"
)

//go:embed *.sql
var sqlFiles embed.FS

func GetMigrations() migrations.MigrationSource {
	builder := migrations.NewMigrationBuilder("gorest-auth")

	builder.Add(
		"20250121000001000",
		"create_users_table",
		func(ctx context.Context, db database.Database) error {
			if err := migrations.SQL(ctx, db, migrations.DialectSQL{
				Postgres: `CREATE TABLE IF NOT EXISTS users (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					email VARCHAR(255) NOT NULL UNIQUE,
					password VARCHAR(255) NOT NULL,
					name VARCHAR(255),
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					deleted_at TIMESTAMP
				)`,
				MySQL: `CREATE TABLE IF NOT EXISTS users (
					id CHAR(36) PRIMARY KEY,
					email VARCHAR(255) NOT NULL UNIQUE,
					password VARCHAR(255) NOT NULL,
					name VARCHAR(255),
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					deleted_at TIMESTAMP NULL,
					INDEX idx_users_email (email),
					INDEX idx_users_deleted_at (deleted_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
				SQLite: `CREATE TABLE IF NOT EXISTS users (
					id TEXT PRIMARY KEY,
					email TEXT NOT NULL UNIQUE,
					password TEXT NOT NULL,
					name TEXT,
					created_at TEXT NOT NULL DEFAULT (datetime('now')),
					updated_at TEXT NOT NULL DEFAULT (datetime('now')),
					deleted_at TEXT
				)`,
			}); err != nil {
				return err
			}

			if db.DriverName() == "postgres" {
				if err := migrations.CreateIndex(ctx, db, "idx_users_email", "users", "email"); err != nil {
					return err
				}
				if err := migrations.CreateIndex(ctx, db, "idx_users_deleted_at", "users", "deleted_at"); err != nil {
					return err
				}

				if err := migrations.SQL(ctx, db, migrations.DialectSQL{
					Postgres: `CREATE OR REPLACE FUNCTION update_updated_at_column()
						RETURNS TRIGGER AS $$
						BEGIN
							NEW.updated_at = CURRENT_TIMESTAMP;
							RETURN NEW;
						END;
						$$ language 'plpgsql'`,
				}); err != nil {
					return err
				}

				return migrations.SQL(ctx, db, migrations.DialectSQL{
					Postgres: `CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
						FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,
				})
			}

			if db.DriverName() == "sqlite" {
				if err := migrations.CreateIndex(ctx, db, "idx_users_email", "users", "email"); err != nil {
					return err
				}
				if err := migrations.CreateIndex(ctx, db, "idx_users_deleted_at", "users", "deleted_at"); err != nil {
					return err
				}

				return migrations.SQL(ctx, db, migrations.DialectSQL{
					SQLite: `CREATE TRIGGER IF NOT EXISTS update_users_updated_at
						AFTER UPDATE ON users
						FOR EACH ROW
						BEGIN
							UPDATE users SET updated_at = datetime('now') WHERE id = NEW.id;
						END`,
				})
			}

			return nil
		},
		func(ctx context.Context, db database.Database) error {
			if db.DriverName() == "postgres" {
				migrations.SQL(ctx, db, migrations.DialectSQL{
					Postgres: `DROP TRIGGER IF EXISTS update_users_updated_at ON users`,
				})
				migrations.SQL(ctx, db, migrations.DialectSQL{
					Postgres: `DROP FUNCTION IF EXISTS update_updated_at_column()`,
				})
				migrations.DropIndex(ctx, db, "idx_users_email", "users")
				migrations.DropIndex(ctx, db, "idx_users_deleted_at", "users")
			}

			if db.DriverName() == "sqlite" {
				migrations.SQL(ctx, db, migrations.DialectSQL{
					SQLite: `DROP TRIGGER IF EXISTS update_users_updated_at`,
				})
				migrations.DropIndex(ctx, db, "idx_users_email", "users")
				migrations.DropIndex(ctx, db, "idx_users_deleted_at", "users")
			}

			return migrations.DropTableIfExists(ctx, db, "users")
		},
	)

	return builder.Build()
}
