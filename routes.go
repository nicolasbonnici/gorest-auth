package auth

import (
	stdcontext "context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest-auth/context"
	"github.com/nicolasbonnici/gorest-auth/middleware"
	"github.com/nicolasbonnici/gorest-auth/models"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/pagination"
	"github.com/nicolasbonnici/gorest/query"
	"github.com/nicolasbonnici/gorest/response"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	Firstname string `json:"firstname" validate:"required"`
	Lastname  string `json:"lastname" validate:"required"`
}

type UpdateUserRequest struct {
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
	Password  *string `json:"password,omitempty" validate:"omitempty,min=8"`
	Firstname *string `json:"firstname,omitempty"`
	Lastname  *string `json:"lastname,omitempty"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func RegisterAuthRoutes(app *fiber.App, db database.Database, jwt *JWTService) {
	authGroup := app.Group("/auth")
	userCRUD := crud.New[models.User](db)

	authGroup.Post("/register", handleRegister(db, userCRUD, jwt))
	authGroup.Post("/login", handleLogin(db, jwt))
	authGroup.Post("/refresh", handleRefresh(jwt))
}

func handleRegister(db database.Database, userCRUD *crud.CRUD[models.User], jwt *JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		ctx := c.Context()

		if err := checkEmailExists(ctx, db, req.Email, uuid.Nil); err != nil {
			return err
		}

		password := req.Password
		user := models.User{
			ID:        uuid.New(),
			Email:     req.Email,
			Password:  &password,
			Firstname: req.Firstname,
			Lastname:  req.Lastname,
			CreatedAt: time.Now(),
		}

		if err := user.HashPassword(); err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to hash password")
		}

		if err := userCRUD.Create(ctx, user); err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to create user")
		}

		token, err := jwt.GenerateToken(user.ID.String())
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to generate token")
		}

		return response.SendCreated(c, AuthResponse{
			Token: token,
			User:  &user,
		})
	}
}

func handleLogin(db database.Database, jwt *JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		ctx := c.Context()

		user, err := getUserByEmail(ctx, db, req.Email)
		if err != nil {
			return err
		}

		if !user.CheckPassword(req.Password) {
			return response.SendError(c, fiber.StatusUnauthorized, "invalid email or password")
		}

		token, err := jwt.GenerateToken(user.ID.String())
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to generate token")
		}

		return response.SendFormatted(c, fiber.StatusOK, AuthResponse{
			Token: token,
			User:  user,
		})
	}
}

func handleRefresh(jwt *JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		type RefreshRequest struct {
			Token string `json:"token" validate:"required"`
		}

		var req RefreshRequest
		if err := c.BodyParser(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		newToken, err := jwt.RefreshToken(req.Token)
		if err != nil {
			return response.SendError(c, fiber.StatusUnauthorized, "invalid or expired token")
		}

		return response.SendFormatted(c, fiber.StatusOK, fiber.Map{
			"token": newToken,
		})
	}
}

func checkEmailExists(ctx stdcontext.Context, db database.Database, email string, excludeUserID uuid.UUID) error {
	qb := query.New(db.Dialect()).
		Select("email").
		From("users").
		Where(query.Eq("email", email))

	if excludeUserID != uuid.Nil {
		qb = qb.Where(query.Ne("id", excludeUserID))
	}

	queryStr, args, err := qb.Build()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	var existingEmail string
	err = db.QueryRow(ctx, queryStr, args...).Scan(&existingEmail)
	if err == nil {
		if excludeUserID == uuid.Nil {
			return fmt.Errorf("user with this email already exists")
		}
		return fmt.Errorf("email already in use")
	}
	if !crud.IsNotFoundError(err) {
		return fmt.Errorf("failed to check existing email: %w", err)
	}

	return nil
}

func getUserByEmail(ctx stdcontext.Context, db database.Database, email string) (*models.User, error) {
	qb := query.New(db.Dialect()).
		Select("id", "firstname", "lastname", "email", "password", "created_at", "updated_at").
		From("users").
		Where(query.Eq("email", email))

	queryStr, args, err := qb.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var user models.User
	var password *string
	var updatedAt *time.Time
	err = db.QueryRow(ctx, queryStr, args...).
		Scan(&user.ID, &user.Firstname, &user.Lastname, &user.Email, &password, &user.CreatedAt, &updatedAt)
	if crud.IsNotFoundError(err) {
		return nil, fmt.Errorf("invalid email or password")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	user.Password = password
	user.UpdatedAt = updatedAt

	return &user, nil
}

func RegisterUserRoutes(app *fiber.App, db database.Database, jwt *JWTService) {
	authMiddleware := middleware.NewAuthMiddleware(jwt)
	userCRUD := crud.New[models.User](db)

	app.Get("/users", authMiddleware, handleListUsers(userCRUD))
	app.Get("/users/:id", authMiddleware, handleGetUser(userCRUD))
	app.Put("/users/:id", authMiddleware, handleUpdateUser(db, userCRUD))
	app.Delete("/users/:id", authMiddleware, handleDeleteUser(userCRUD))
}

func handleListUsers(userCRUD *crud.CRUD[models.User]) fiber.Handler {
	return func(c *fiber.Ctx) error {
		const defaultLimit = 10
		const maxLimit = 100

		limit := pagination.ParseIntQuery(c, "limit", defaultLimit, maxLimit)
		page := max(1, pagination.ParseIntQuery(c, "page", 1, 1000))
		offset := (page - 1) * limit

		ctx := c.Context()

		result, err := userCRUD.GetAllPaginated(ctx, crud.PaginationOptions{
			Limit:        limit,
			Offset:       offset,
			IncludeCount: true,
			OrderBy: []crud.OrderByClause{
				{Column: "created_at", Direction: query.DESC},
			},
		})
		if err != nil {
			return pagination.SendPaginatedError(c, fiber.StatusInternalServerError, "failed to fetch users")
		}

		return pagination.SendHydraCollection(c, result.Items, result.Total, limit, page, defaultLimit)
	}
}

func handleGetUser(userCRUD *crud.CRUD[models.User]) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := parseUserIDParam(c)
		if err != nil {
			return err
		}

		ctx := c.Context()

		user, err := userCRUD.GetByID(ctx, userID)
		if crud.IsNotFoundError(err) {
			return response.SendError(c, fiber.StatusNotFound, "user not found")
		}
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "database error")
		}

		return response.SendFormatted(c, fiber.StatusOK, user)
	}
}

func handleUpdateUser(db database.Database, userCRUD *crud.CRUD[models.User]) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := parseUserIDParam(c)
		if err != nil {
			return err
		}

		if err := checkUserOwnership(c, userID); err != nil {
			return err
		}

		var req UpdateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		ctx := c.Context()

		user, err := userCRUD.GetByID(ctx, userID)
		if crud.IsNotFoundError(err) {
			return response.SendError(c, fiber.StatusNotFound, "user not found")
		}
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "database error")
		}

		if err := updateUserFields(ctx, db, user, &req, userID); err != nil {
			return err
		}

		now := time.Now()
		user.UpdatedAt = &now

		if err := userCRUD.Update(ctx, userID, *user); err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to update user")
		}

		user.Password = nil

		return response.SendFormatted(c, fiber.StatusOK, user)
	}
}

func handleDeleteUser(userCRUD *crud.CRUD[models.User]) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := parseUserIDParam(c)
		if err != nil {
			return err
		}

		if err := checkUserOwnership(c, userID); err != nil {
			return err
		}

		ctx := c.Context()

		_, err = userCRUD.GetByID(ctx, userID)
		if crud.IsNotFoundError(err) {
			return response.SendError(c, fiber.StatusNotFound, "user not found")
		}
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "database error")
		}

		if err := userCRUD.Delete(ctx, userID); err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to delete user")
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

func parseUserIDParam(c *fiber.Ctx) (uuid.UUID, error) {
	id := c.Params("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, response.SendError(c, fiber.StatusBadRequest, "invalid user id")
	}
	return userID, nil
}

func checkUserOwnership(c *fiber.Ctx, userID uuid.UUID) error {
	authenticatedUserID, ok := context.GetUserID(c)
	if !ok {
		return response.SendError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	if authenticatedUserID != userID.String() {
		return response.SendError(c, fiber.StatusForbidden, "you can only modify your own account")
	}

	return nil
}

func updateUserFields(ctx stdcontext.Context, db database.Database, user *models.User, req *UpdateUserRequest, userID uuid.UUID) error {
	if req.Email != nil {
		if err := checkEmailExists(ctx, db, *req.Email, userID); err != nil {
			return err
		}
		user.Email = *req.Email
	}

	if req.Firstname != nil {
		user.Firstname = *req.Firstname
	}

	if req.Lastname != nil {
		user.Lastname = *req.Lastname
	}

	if req.Password != nil {
		user.Password = req.Password
		if err := user.HashPassword(); err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
	}

	return nil
}
