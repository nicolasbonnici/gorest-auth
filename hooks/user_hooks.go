package hooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/nicolasbonnici/gorest-auth/models"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/hooks"
	"github.com/nicolasbonnici/gorest/rbac"
)

type UserHooks struct {
	*hooks.DefaultAuthorization[models.User]
	hooks.NoOpHooks[models.User]
	db database.Database
}

func NewUserHooks(db database.Database, config rbac.Config) *UserHooks {
	return &UserHooks{
		DefaultAuthorization: hooks.NewDefaultAuthorization[models.User](config),
		NoOpHooks:            *hooks.NewNoOpHooks[models.User](),
		db:                   db,
	}
}

func (h *UserHooks) CheckUpdate(ctx context.Context, id any, model *models.User) error {
	userID, hasUserID := rbac.GetUserID(ctx)
	roles, _ := rbac.GetRoles(ctx)

	if h.GetVoter().IsSuperuser(roles) {
		return nil
	}

	if hasUserID {
		recordID := fmt.Sprintf("%v", id)
		if recordID == userID {
			return nil
		}
	}

	return rbac.ErrPermissionDenied
}

func (h *UserHooks) StateProcessor(ctx context.Context, op hooks.Operation, id any, model *models.User) error {
	switch op {
	case hooks.OperationCreate, hooks.OperationUpdate:
		if model.Email != "" && !strings.Contains(model.Email, "@") {
			return fmt.Errorf("invalid email format")
		}

		if model.Password != nil && *model.Password != "" {
			if len(*model.Password) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}
			if err := model.HashPassword(); err != nil {
				return err
			}
		}

		if model.Role != "" && model.Role != "user" && model.Role != "admin" {
			return fmt.Errorf("invalid role: must be 'user' or 'admin'")
		}
	}

	return nil
}
