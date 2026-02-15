# GoREST Auth Plugin

[![CI](https://github.com/nicolasbonnici/gorest-auth/actions/workflows/ci.yml/badge.svg)](https://github.com/nicolasbonnici/gorest-auth/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nicolasbonnici/gorest-auth)](https://goreportcard.com/report/github.com/nicolasbonnici/gorest-auth)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A production-ready JWT-based authentication plugin for GoREST 0.4+ with built-in user management, bcrypt password hashing, and automatic migration support.

## Features

- **JWT Authentication**: Secure token-based authentication with configurable TTL
- **User Management**: Complete user registration and login system
- **Password Security**: Bcrypt password hashing with industry-standard cost factor
- **Built-in Migrations**: Automatic database schema management for PostgreSQL, MySQL, and SQLite
- **Context Helpers**: Easy access to authenticated user ID in request handlers
- **Token Refresh**: Built-in token refresh endpoint for seamless session management
- **Multi-Database Support**: Compatible with PostgreSQL, MySQL, and SQLite
- **Middleware Integration**: Plug-and-play middleware for protecting routes

## Installation

```bash
go get github.com/nicolasbonnici/gorest-auth
```

## Requirements

- Go 1.25.1+
- GoREST 0.4+ (with migration support)
- PostgreSQL, MySQL, or SQLite database


## Development Environment

To set up your development environment:

```bash
make install
```

This will:
- Install Go dependencies
- Install development tools (golangci-lint)
- Set up git hooks (pre-commit linting and tests)

## Quick Start

### Basic Setup

```go
package main

import (
    "github.com/nicolasbonnici/gorest"
    "github.com/nicolasbonnici/gorest/pluginloader"

    authplugin "github.com/nicolasbonnici/gorest-auth"
)

func init() {
    // Register auth plugin
    pluginloader.RegisterPluginFactory("auth", authplugin.NewPlugin)
}

func main() {
    cfg := gorest.Config{
        ConfigPath: ".",
    }

    gorest.Start(cfg)
}
```

### Configuration (gorest.yaml)

```yaml
database:
  url: "${DATABASE_URL}"

plugins:
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 900  # 15 minutes (in seconds)

# Migration configuration (GoREST 0.4+)
migrations:
  enabled: true
  auto_migrate: true  # Run migrations on startup
```

### Environment Variables

Create a `.env` file:

```env
DATABASE_URL="postgres://user:password@localhost:5432/mydb?sslmode=disable"
JWT_SECRET="your-super-secret-jwt-key-minimum-32-characters-long"
```

**Important**: Use a strong, random secret for `JWT_SECRET` in production.

## API Endpoints

The plugin automatically registers the following endpoints:

### Register a New User

```bash
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123",
  "name": "John Doe"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "name": "John Doe",
    "created_at": "2025-01-21T10:30:00Z",
    "updated_at": "2025-01-21T10:30:00Z"
  }
}
```

### Login

```bash
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "name": "John Doe",
    "created_at": "2025-01-21T10:30:00Z",
    "updated_at": "2025-01-21T10:30:00Z"
  }
}
```

### Refresh Token

```bash
POST /auth/refresh
Content-Type: application/json

{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

## Using the Auth Middleware

### Protecting Routes

The plugin provides a middleware that you can use to protect specific routes:

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest-auth/context"
)

func setupRoutes(app *fiber.App, pluginRegistry *plugin.Registry) {
    // Get auth middleware from plugin
    var authMiddleware fiber.Handler
    if authPlugin, ok := pluginRegistry.Get("auth"); ok {
        authMiddleware = authPlugin.Handler()
    }

    // Public route - no authentication required
    app.Get("/public", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "message": "This is a public endpoint",
        })
    })

    // Protected route - authentication required
    app.Get("/protected", authMiddleware, func(c *fiber.Ctx) error {
        // Get authenticated user ID from context
        userID := context.MustGetUserID(c)

        return c.JSON(fiber.Map{
            "message": "This is a protected endpoint",
            "user_id": userID,
        })
    })

    // Protected group - all routes require authentication
    api := app.Group("/api", authMiddleware)

    api.Get("/profile", func(c *fiber.Ctx) error {
        userID, ok := context.GetUserID(c)
        if !ok {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "user not authenticated",
            })
        }

        return c.JSON(fiber.Map{
            "user_id": userID,
        })
    })
}
```

### Making Authenticated Requests

Include the JWT token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
     http://localhost:8000/protected
```

## Context Helpers

The plugin provides convenient context helpers to access authenticated user information:

```go
import "github.com/nicolasbonnici/gorest-auth/context"

// Get user ID (returns userID and boolean indicating if found)
userID, ok := context.GetUserID(c)
if !ok {
    // User not authenticated
}

// Get user ID (returns empty string if not found)
userID := context.MustGetUserID(c)
```

## Database Schema

The plugin creates a `users` table with the following structure:

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key (auto-generated) |
| `email` | VARCHAR(255) | User email (unique) |
| `password` | VARCHAR(255) | Bcrypt-hashed password |
| `name` | VARCHAR(255) | User's display name |
| `created_at` | TIMESTAMP | Account creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |
| `deleted_at` | TIMESTAMP | Soft delete timestamp (nullable) |

**Indexes:**
- Unique index on `email`
- Index on `deleted_at` for soft delete queries

## Migration System

The auth plugin uses GoREST 0.4's migration system with support for multiple databases.

### Automatic Migration on Startup

Migrations run automatically when `migrations.auto_migrate: true` is set in your configuration.

### Manual Migration Control

```bash
# Run pending migrations
gorest migrate up

# Run migrations for auth plugin specifically
gorest migrate up --source auth

# Rollback last migration
gorest migrate down

# Check migration status
gorest migrate status
```

### Migration Files

The plugin includes migrations for all supported databases:
- `migrations/20250121000001_create_users_table.up.postgres.sql`
- `migrations/20250121000001_create_users_table.up.mysql.sql`
- `migrations/20250121000001_create_users_table.up.sqlite.sql`

## Advanced Usage

### Custom JWT TTL

Configure token expiration time in seconds:

```yaml
plugins:
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 3600  # 1 hour
```

### Accessing the User Model

```go
import (
    "github.com/nicolasbonnici/gorest-auth/models"
    "github.com/nicolasbonnici/gorest/database"
)

func getUser(db database.Database, userID string) (*models.User, error) {
    var user models.User
    if err := db.GetDB().Where("id = ?", userID).First(&user).Error; err != nil {
        return nil, err
    }
    return &user, nil
}
```

### Password Hashing

The `User` model includes built-in password hashing methods:

```go
user := &models.User{
    Email:    "user@example.com",
    Password: "plaintextpassword",
}

// Hash password before saving
if err := user.HashPassword(); err != nil {
    // Handle error
}

// Verify password during login
if !user.CheckPassword("plaintextpassword") {
    // Invalid password
}
```

## Security Best Practices

### JWT Secret

- **Never** commit your JWT secret to version control
- Use a strong, random secret (minimum 32 characters)
- Generate a secure secret:
  ```bash
  openssl rand -base64 32
  ```

### Password Requirements

The plugin requires passwords to be at least 8 characters. You can add additional validation:

```go
import "github.com/go-playground/validator/v10"

type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8,max=72"`
    Name     string `json:"name" validate:"required,min=2"`
}
```

### HTTPS in Production

Always use HTTPS in production to prevent token interception:

```go
app := fiber.New(fiber.Config{
    DisableStartupMessage: false,
})

// Use TLS
app.ListenTLS(":443", "cert.pem", "key.pem")
```

### Token Storage (Client-Side)

**Recommended approaches:**
- Use `httpOnly` cookies for web applications
- Use secure storage (Keychain/Keystore) for mobile apps
- Avoid localStorage for sensitive tokens

## Error Handling

The plugin returns standard HTTP status codes:

| Status Code | Meaning |
|-------------|---------|
| `201` | User successfully registered |
| `200` | Login successful / Token refreshed |
| `400` | Invalid request body |
| `401` | Invalid credentials / Expired token |
| `409` | User already exists (registration) |
| `500` | Internal server error |

**Example error response:**
```json
{
  "error": "invalid email or password"
}
```

## Integration with Other Plugins

The auth plugin is designed to work seamlessly with other GoREST plugins:

```go
import (
    authplugin "github.com/nicolasbonnici/gorest-auth"
    blog "github.com/nicolasbonnici/gorest-blog-plugin"
)

func init() {
    // Register auth plugin first
    pluginloader.RegisterPluginFactory("auth", authplugin.NewPlugin)

    // Other plugins can depend on auth
    pluginloader.RegisterPluginFactory("blog", blog.NewPlugin)
}
```

The blog plugin declares auth as a dependency, ensuring the users table exists before creating posts.

## Testing

### Manual Testing with curl

**Register:**
```bash
curl -X POST http://localhost:8000/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpass123",
    "name": "Test User"
  }'
```

**Login:**
```bash
curl -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpass123"
  }'
```

**Access Protected Route:**
```bash
TOKEN="your-jwt-token-here"

curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8000/protected
```

## Project Structure

```
gorest-auth/
├── plugin.go              # Main plugin implementation
├── config.go              # Configuration structure
├── jwt.go                 # JWT token generation and validation
├── routes.go              # Auth endpoint handlers (register, login, refresh)
├── go.mod                 # Go module definition
├── README.md              # This file
├── migrations/            # Database migrations
│   ├── 20250121000001_create_users_table.up.postgres.sql
│   ├── 20250121000001_create_users_table.down.postgres.sql
│   ├── 20250121000001_create_users_table.up.mysql.sql
│   ├── 20250121000001_create_users_table.down.mysql.sql
│   ├── 20250121000001_create_users_table.up.sqlite.sql
│   └── 20250121000001_create_users_table.down.sqlite.sql
├── models/                # Data models
│   └── user.go
├── middleware/            # HTTP middleware
│   └── auth.go
└── context/               # Context helpers
    └── auth.go
```

## Troubleshooting

### "missing authorization header"

Ensure you're including the `Authorization` header with the `Bearer` prefix:
```
Authorization: Bearer <your-token>
```

### "invalid or expired token"

- Check if the token has expired (default TTL is 15 minutes)
- Verify the `JWT_SECRET` matches between token generation and validation
- Use the `/auth/refresh` endpoint to get a new token

### Migration Errors

**Problem:** `migration failed: relation "users" already exists`

**Solution:** The users table was created outside migrations. Either:
1. Drop the table and re-run migrations, or
2. Mark the migration as applied: `gorest migrate force`

### "user with this email already exists"

This is a constraint violation. Check if the user exists before registration, or use the login endpoint instead.

## Performance Considerations

- **Password Hashing**: Uses bcrypt with default cost (10), balancing security and performance
- **Token Validation**: JWT validation is fast (< 1ms) with proper secret configuration
- **Database Indexes**: Email lookups are optimized with a unique index
- **Connection Pooling**: Relies on GoREST's database connection pool

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

---

## Git Hooks

This directory contains git hooks for the GoREST plugin to maintain code quality.

### Available Hooks

#### pre-commit

Runs before each commit to ensure code quality:
- **Linting**: Runs `make lint` to check code style and potential issues
- **Tests**: Runs `make test` to verify all tests pass

### Installation

#### Automatic Installation

Run the install script from the project root:

```bash
./.githooks/install.sh
```

#### Manual Installation

Copy the hooks to your `.git/hooks` directory:

```bash
cp .githooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

---


## License

MIT License - See LICENSE file for details

## Changelog

### v1.0.0 (2025-01-21)
- Initial release
- JWT-based authentication
- User registration and login
- Token refresh endpoint
- PostgreSQL, MySQL, SQLite support
- GoREST 0.4 migration system integration
- Bcrypt password hashing
- Context helpers for user ID access
- Auth middleware for route protection

## Support

For issues, questions, or contributions, please visit:
- GitHub Issues: https://github.com/nicolasbonnici/gorest-auth/issues
- GoREST Documentation: https://github.com/nicolasbonnici/gorest

## Related Projects

- [GoREST](https://github.com/nicolasbonnici/gorest) - The main GoREST framework
- [GoREST Blog Plugin](https://github.com/nicolasbonnici/gorest-blog-plugin) - Blog plugin with auth integration
