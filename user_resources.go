package auth

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest-auth/converters"
	"github.com/nicolasbonnici/gorest-auth/dtos"
	"github.com/nicolasbonnici/gorest-auth/hooks"
	"github.com/nicolasbonnici/gorest-auth/middleware"
	"github.com/nicolasbonnici/gorest-auth/models"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/filter"
	"github.com/nicolasbonnici/gorest/pagination"
	"github.com/nicolasbonnici/gorest/query"
	"github.com/nicolasbonnici/gorest/response"
)

type UserResource struct {
	db        database.Database
	crud      *crud.CRUD[models.User]
	hooks     *hooks.UserHooks
	converter *converters.UserConverter
}

func RegisterUserRoutes(router fiber.Router, db database.Database, jwt *JWTService) {
	authMiddleware := middleware.NewAuthMiddleware(jwt)

	resource := &UserResource{
		db:        db,
		crud:      crud.New[models.User](db),
		hooks:     hooks.NewUserHooks(db),
		converter: &converters.UserConverter{},
	}

	router.Get("/users", authMiddleware, resource.GetAll)
	router.Get("/users/:id", authMiddleware, resource.GetByID)
	router.Put("/users/:id", authMiddleware, resource.Update)
	router.Delete("/users/:id", authMiddleware, resource.Delete)
}

func (r *UserResource) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, "invalid user ID")
	}

	if err := r.hooks.GetByIDHook(c, id); err != nil {
		return err
	}

	user, err := r.crud.GetByID(c.Context(), id)
	if crud.IsNotFoundError(err) {
		return response.SendError(c, fiber.StatusNotFound, "user not found")
	}
	if err != nil {
		return response.SendError(c, fiber.StatusInternalServerError, "database error")
	}

	return response.SendFormatted(c, fiber.StatusOK, r.converter.ModelToResponseDTO(*user))
}

func (r *UserResource) GetAll(c *fiber.Ctx) error {
	limit := pagination.ParseIntQuery(c, "limit", 20, 100)
	page := pagination.ParseIntQuery(c, "page", 1, 10000)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	includeCount := c.Query("count", "true") != "false"

	queryParams := make(url.Values)
	for key, value := range c.Context().QueryArgs().All() {
		queryParams.Add(string(key), string(value))
	}

	fieldMap := map[string]string{
		"id":         "id",
		"email":      "email",
		"firstname":  "firstname",
		"lastname":   "lastname",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}

	var conditions []query.Condition
	filters := filter.NewFilterSetWithMapping(fieldMap, r.db.Dialect())
	if err := filters.ParseFromQuery(queryParams); err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}
	conditions = filters.Conditions()

	var orderBy []crud.OrderByClause
	ordering := filter.NewOrderSetWithMapping(fieldMap)
	if err := ordering.ParseFromQuery(queryParams); err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	orderClauses := ordering.OrderClauses()
	orderBy = make([]crud.OrderByClause, len(orderClauses))
	for i, oc := range orderClauses {
		orderBy[i] = crud.OrderByClause{
			Column:    oc.Column,
			Direction: oc.Direction,
		}
	}

	if err := r.hooks.GetAllHook(c, &conditions, &orderBy); err != nil {
		return err
	}

	result, err := r.crud.GetAllPaginated(c.Context(), crud.PaginationOptions{
		Limit:        limit,
		Offset:       offset,
		IncludeCount: includeCount,
		Conditions:   conditions,
		OrderBy:      orderBy,
	})
	if err != nil {
		return response.SendError(c, fiber.StatusInternalServerError, "database error")
	}

	return pagination.SendHydraCollection(c, r.converter.ModelsToResponseDTOs(result.Items), result.Total, limit, page, 20)
}

func (r *UserResource) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var dto dtos.UserUpdateDTO
	if err := c.BodyParser(&dto); err != nil {
		return response.SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	model := r.converter.UpdateDTOToModel(dto)

	if err := r.hooks.UpdateHook(c, dto, &model); err != nil {
		return err
	}

	if err := r.crud.Update(c.Context(), id, model); err != nil {
		if crud.IsNotFoundError(err) {
			return response.SendError(c, fiber.StatusNotFound, "user not found")
		}
		return response.SendError(c, fiber.StatusInternalServerError, "database error")
	}

	dto2 := r.converter.ModelToResponseDTO(model)
	return response.SendFormatted(c, fiber.StatusOK, dto2)
}

func (r *UserResource) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := r.hooks.DeleteHook(c, id); err != nil {
		return err
	}

	if err := r.crud.Delete(c.Context(), id); err != nil {
		if crud.IsNotFoundError(err) {
			return response.SendError(c, fiber.StatusNotFound, "user not found")
		}
		return response.SendError(c, fiber.StatusInternalServerError, "database error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
