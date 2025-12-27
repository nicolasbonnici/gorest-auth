package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Firstname string     `json:"firstname" gorm:"not null"`
	Lastname  string     `json:"lastname" gorm:"not null"`
	Email     string     `json:"email" gorm:"uniqueIndex;not null"`
	Password  *string    `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) HashPassword() error {
	if u.Password == nil || *u.Password == "" {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	hashedStr := string(hashedPassword)
	u.Password = &hashedStr
	return nil
}

func (u *User) CheckPassword(password string) bool {
	if u.Password == nil {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(*u.Password), []byte(password))
	return err == nil
}
