package auth

import (
	"context"

	"github.com/gofiber/fiber/v2"
	authcontext "github.com/nicolasbonnici/gorest-auth/context"
)

type AuthenticatedUser struct {
	UserID string
}

// This is a compatibility function for the codegen
func Context(c *fiber.Ctx) context.Context {
	return c.Context()
}

// This is a compatibility function for the codegen
func GetAuthenticatedUser(c *fiber.Ctx) *AuthenticatedUser {
	userID, ok := authcontext.GetUserID(c)
	if !ok {
		return nil
	}

	return &AuthenticatedUser{
		UserID: userID,
	}
}
