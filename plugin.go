package auth

import (
	"embed"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest-auth/middleware"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/migrations"
	"github.com/nicolasbonnici/gorest/plugin"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type AuthPlugin struct {
	config Config
	db     database.Database
	jwt    *JWTService
}

func NewPlugin() plugin.Plugin {
	return &AuthPlugin{}
}

func (p *AuthPlugin) Name() string {
	return "auth"
}

func (p *AuthPlugin) Initialize(config map[string]interface{}) error {
	p.config = DefaultConfig()

	if db, ok := config["database"].(database.Database); ok {
		p.db = db
		p.config.Database = db
	}

	if jwtSecret, ok := config["jwt_secret"].(string); ok {
		p.config.JWTSecret = jwtSecret
	}

	if jwtTTL, ok := config["jwt_ttl"].(int); ok {
		p.config.JWTTTL = jwtTTL
	}

	p.jwt = NewJWTService(p.config.JWTSecret, p.config.JWTTTL)

	return nil
}

func (p *AuthPlugin) Handler() fiber.Handler {
	return middleware.NewAuthMiddleware(p.jwt)
}

func (p *AuthPlugin) SetupEndpoints(app *fiber.App) error {
	if p.db == nil {
		return nil
	}

	RegisterAuthRoutes(app, p.db, p.jwt)
	return nil
}

func (p *AuthPlugin) MigrationSource() interface{} {
	return migrations.NewEmbeddedSource("auth", migrationFiles, "migrations", p.db)
}

func (p *AuthPlugin) MigrationDependencies() []string {
	return []string{}
}
