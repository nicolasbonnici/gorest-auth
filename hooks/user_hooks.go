package hooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest-auth/dtos"
	"github.com/nicolasbonnici/gorest-auth/models"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/query"
)

type UserHooks struct {
	db database.Database
}

func NewUserHooks(db database.Database) *UserHooks {
	return &UserHooks{db: db}
}

func (h *UserHooks) CreateHook(c *fiber.Ctx, dto dtos.UserCreateDTO, model *models.User) error {
	if err := h.validateEmail(dto.Email); err != nil {
		return fiber.NewError(400, err.Error())
	}

	if len(dto.Password) < 8 {
		return fiber.NewError(400, "password must be at least 8 characters")
	}

	if dto.Firstname == "" {
		return fiber.NewError(400, "firstname is required")
	}

	if dto.Lastname == "" {
		return fiber.NewError(400, "lastname is required")
	}

	if err := h.checkEmailExists(c.Context(), dto.Email, uuid.Nil); err != nil {
		return err
	}

	if err := model.HashPassword(); err != nil {
		return fiber.NewError(500, "failed to hash password")
	}

	return nil
}

func (h *UserHooks) UpdateHook(c *fiber.Ctx, dto dtos.UserUpdateDTO, model *models.User) error {
	id := c.Params("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		return fiber.NewError(400, "invalid user ID")
	}

	if dto.Email != nil {
		if err := h.validateEmail(*dto.Email); err != nil {
			return fiber.NewError(400, err.Error())
		}

		if err := h.checkEmailExists(c.Context(), *dto.Email, userID); err != nil {
			return err
		}
	}

	if dto.Password != nil {
		if len(*dto.Password) < 8 {
			return fiber.NewError(400, "password must be at least 8 characters")
		}

		if err := model.HashPassword(); err != nil {
			return fiber.NewError(500, "failed to hash password")
		}
	}

	return nil
}

func (h *UserHooks) DeleteHook(c *fiber.Ctx, id any) error {
	return nil
}

func (h *UserHooks) GetByIDHook(c *fiber.Ctx, id any) error {
	return nil
}

func (h *UserHooks) GetAllHook(c *fiber.Ctx, conditions *[]query.Condition, orderBy *[]crud.OrderByClause) error {
	return nil
}

func (h *UserHooks) validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func (h *UserHooks) checkEmailExists(ctx context.Context, email string, excludeUserID uuid.UUID) error {
	sql := "SELECT COUNT(*) FROM users WHERE email = " + h.db.Dialect().Placeholder(1)
	args := []any{email}

	if excludeUserID != uuid.Nil {
		sql += " AND id != " + h.db.Dialect().Placeholder(2)
		args = append(args, excludeUserID)
	}

	var count int
	if err := h.db.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return fiber.NewError(500, "database error")
	}

	if count > 0 {
		return fiber.NewError(409, "email already exists")
	}

	return nil
}
