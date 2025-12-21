package auth

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest-auth/models"
	"github.com/nicolasbonnici/gorest/database"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func RegisterAuthRoutes(app *fiber.App, db database.Database, jwt *JWTService) {
	authGroup := app.Group("/auth")

	authGroup.Post("/register", func(c *fiber.Ctx) error {
		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		ctx := c.Context()

		var existingEmail string
		err := db.QueryRow(ctx, "SELECT email FROM users WHERE email = $1 AND deleted_at IS NULL", req.Email).Scan(&existingEmail)
		if err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "user with this email already exists",
			})
		}
		if err != nil && err != sql.ErrNoRows {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check existing user",
			})
		}

		user := &models.User{
			ID:        uuid.New(),
			Email:     req.Email,
			Password:  req.Password,
			Name:      req.Name,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := user.HashPassword(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to hash password",
			})
		}

		_, err = db.Exec(ctx, "INSERT INTO users (id, email, password, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
			user.ID, user.Email, user.Password, user.Name, user.CreatedAt, user.UpdatedAt)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create user",
			})
		}

		token, err := jwt.GenerateToken(user.ID.String())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to generate token",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(AuthResponse{
			Token: token,
			User:  user,
		})
	})

	authGroup.Post("/login", func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		ctx := c.Context()

		var user models.User
		var deletedAt sql.NullTime
		err := db.QueryRow(ctx, "SELECT id, email, password, name, created_at, updated_at, deleted_at FROM users WHERE email = $1", req.Email).
			Scan(&user.ID, &user.Email, &user.Password, &user.Name, &user.CreatedAt, &user.UpdatedAt, &deletedAt)
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid email or password",
			})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "database error",
			})
		}

		if deletedAt.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid email or password",
			})
		}

		if !user.CheckPassword(req.Password) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid email or password",
			})
		}

		token, err := jwt.GenerateToken(user.ID.String())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to generate token",
			})
		}

		return c.JSON(AuthResponse{
			Token: token,
			User:  &user,
		})
	})

	authGroup.Post("/refresh", func(c *fiber.Ctx) error {
		type RefreshRequest struct {
			Token string `json:"token" validate:"required"`
		}

		var req RefreshRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		newToken, err := jwt.RefreshToken(req.Token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		return c.JSON(fiber.Map{
			"token": newToken,
		})
	})
}
