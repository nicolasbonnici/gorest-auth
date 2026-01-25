package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest-auth/context"
	"github.com/nicolasbonnici/gorest-auth/middleware"
	"github.com/nicolasbonnici/gorest-auth/models"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
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

	authGroup.Post("/register", func(c *fiber.Ctx) error {
		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		ctx := c.Context()

		qb := query.New(db.Dialect()).
			Select("email").
			From("users").
			Where(query.Eq("email", req.Email))

		queryStr, args, err := qb.Build()
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to build query")
		}

		var existingEmail string
		err = db.QueryRow(ctx, queryStr, args...).Scan(&existingEmail)
		if err == nil {
			return response.SendError(c, fiber.StatusConflict, "user with this email already exists")
		}
		if !crud.IsNotFoundError(err) {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to check existing user")
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
	})

	authGroup.Post("/login", func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		ctx := c.Context()

		qb := query.New(db.Dialect()).
			Select("id", "firstname", "lastname", "email", "password", "created_at", "updated_at").
			From("users").
			Where(query.Eq("email", req.Email))

		queryStr, args, err := qb.Build()
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to build query")
		}

		var user models.User
		var password *string
		var updatedAt *time.Time
		err = db.QueryRow(ctx, queryStr, args...).
			Scan(&user.ID, &user.Firstname, &user.Lastname, &user.Email, &password, &user.CreatedAt, &updatedAt)
		if crud.IsNotFoundError(err) {
			return response.SendError(c, fiber.StatusUnauthorized, "invalid email or password")
		}
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "database error")
		}

		user.Password = password
		user.UpdatedAt = updatedAt

		if !user.CheckPassword(req.Password) {
			return response.SendError(c, fiber.StatusUnauthorized, "invalid email or password")
		}

		token, err := jwt.GenerateToken(user.ID.String())
		if err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to generate token")
		}

		return response.SendFormatted(c, fiber.StatusOK, AuthResponse{
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
			return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
		}

		newToken, err := jwt.RefreshToken(req.Token)
		if err != nil {
			return response.SendError(c, fiber.StatusUnauthorized, "invalid or expired token")
		}

		return response.SendFormatted(c, fiber.StatusOK, fiber.Map{
			"token": newToken,
		})
	})
}

func RegisterUserRoutes(app *fiber.App, db database.Database, jwt *JWTService) {
	authMiddleware := middleware.NewAuthMiddleware(jwt)
	userCRUD := crud.New[models.User](db)

	app.Get("/users", authMiddleware, func(c *fiber.Ctx) error {
		limit := 10
		if l := c.QueryInt("limit", 10); l > 0 && l <= 100 {
			limit = l
		}
		page := c.QueryInt("page", 1)
		if page < 1 {
			page = 1
		}
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
			return response.SendError(c, fiber.StatusInternalServerError, "failed to fetch users")
		}

		users := make([]*models.User, len(result.Items))
		for i := range result.Items {
			users[i] = &result.Items[i]
		}

		total := int64(0)
		if result.Total != nil {
			total = int64(*result.Total)
		}

		return response.SendFormatted(c, fiber.StatusOK, fiber.Map{
			"data":  users,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	})

	app.Get("/users/:id", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		userID, err := uuid.Parse(id)
		if err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid user id")
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
	})

	app.Put("/users/:id", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		userID, err := uuid.Parse(id)
		if err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid user id")
		}

		authenticatedUserID, ok := context.GetUserID(c)
		if !ok {
			return response.SendError(c, fiber.StatusUnauthorized, "unauthorized")
		}

		if authenticatedUserID != userID.String() {
			return response.SendError(c, fiber.StatusForbidden, "you can only update your own account")
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

		if req.Email != nil {
			qb := query.New(db.Dialect()).
				Select("email").
				From("users").
				Where(query.And(
					query.Eq("email", *req.Email),
					query.Ne("id", userID),
				))

			queryStr, args, err := qb.Build()
			if err != nil {
				return response.SendError(c, fiber.StatusInternalServerError, "failed to build query")
			}

			var existingEmail string
			err = db.QueryRow(ctx, queryStr, args...).Scan(&existingEmail)
			if err == nil {
				return response.SendError(c, fiber.StatusConflict, "email already in use")
			}
			if !crud.IsNotFoundError(err) {
				return response.SendError(c, fiber.StatusInternalServerError, "failed to check existing email")
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
				return response.SendError(c, fiber.StatusInternalServerError, "failed to hash password")
			}
		}

		now := time.Now()
		user.UpdatedAt = &now

		if err := userCRUD.Update(ctx, userID, *user); err != nil {
			return response.SendError(c, fiber.StatusInternalServerError, "failed to update user")
		}

		user.Password = nil

		return response.SendFormatted(c, fiber.StatusOK, user)
	})

	app.Delete("/users/:id", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		userID, err := uuid.Parse(id)
		if err != nil {
			return response.SendError(c, fiber.StatusBadRequest, "invalid user id")
		}

		authenticatedUserID, ok := context.GetUserID(c)
		if !ok {
			return response.SendError(c, fiber.StatusUnauthorized, "unauthorized")
		}

		if authenticatedUserID != userID.String() {
			return response.SendError(c, fiber.StatusForbidden, "you can only delete your own account")
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
	})
}
