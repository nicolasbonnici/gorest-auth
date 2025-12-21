package auth

import "github.com/nicolasbonnici/gorest/database"

type Config struct {
	Database  database.Database
	JWTSecret string
	JWTTTL    int
}

func DefaultConfig() Config {
	return Config{
		JWTTTL: 900,
	}
}
