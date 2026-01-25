package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest-auth/middleware"
	authmigrations "github.com/nicolasbonnici/gorest-auth/migrations"
	"github.com/nicolasbonnici/gorest-auth/models"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

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
	RegisterUserRoutes(app, p.db, p.jwt)
	return nil
}

func (p *AuthPlugin) MigrationSource() interface{} {
	return authmigrations.GetMigrations()
}

func (p *AuthPlugin) MigrationDependencies() []string {
	return []string{}
}

func (p *AuthPlugin) GetOpenAPIResources() []plugin.OpenAPIResource {
	return []plugin.OpenAPIResource{{
		Name:          "user",
		PluralName:    "users",
		BasePath:      "/users",
		Tags:          []string{"Users"},
		ResponseModel: models.User{},
		CreateModel:   RegisterRequest{},
		UpdateModel:   UpdateUserRequest{},
		Description:   "User management and authentication",
	}}
}
