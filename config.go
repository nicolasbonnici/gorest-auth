package auth

import (
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/rbac"
)

type Config struct {
	Database    database.Database
	JWTSecret   string
	JWTTTL      int
	CORSOrigins string
}

func DefaultConfig() Config {
	return Config{
		JWTTTL:      900,
		CORSOrigins: "*",
	}
}

func GetRBACConfig() rbac.Config {
	return rbac.Config{
		DefaultPolicy:      rbac.DenyAll,
		SuperuserRole:      "admin",
		DefaultFieldPolicy: "deny",
		StrictValidation:   false,
		CacheEnabled:       true,
		CacheTTL:           300,
		RoleHierarchy: map[string][]string{
			"admin": {"user"},
		},
	}
}
