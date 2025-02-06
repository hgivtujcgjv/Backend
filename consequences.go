package main

import (
	"database/sql"
	"net/http"
	"time"
)

type SessionManager interface {
	Check(*http.Request) (*Session, error)
	Create(http.ResponseWriter, *User) error
	DestroyAll(*User) error
	DestroyCurrent(http.ResponseWriter, *http.Request) error
}

type Session struct {
	UserID   int
	Username string
}

type UserResponse struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Bio      string `json:"bio"`
	Image    string `json:"image"`
	Token    string `json:"token"`
}

type User struct {
	Id       int
	Email    string
	Password string
	Username string
	Bio      string
	Image    string
}
type StDb struct {
	db *sql.DB
}

func NewSessionsDB(db *sql.DB) *StDb {
	return &StDb{
		db: db,
	}
}

type Article struct {
	Author         UserResponse `json:"author"`
	Body           string       `json:"body"`
	CreatedAt      time.Time    `json:"createdAt"`
	Description    string       `json:"description"`
	Favorited      bool         `json:"favorited"`
	FavoritesCount int          `json:"favoritesCount"`
	Slug           string       `json:"slug"`
	TagList        []string     `json:"tagList"`
	Title          string       `json:"title"`
	UpdatedAt      time.Time    `json:"updatedAt"`
}

type UserHandler struct {
	Bd *sql.DB
	// в будующем тут еще нужен менеджер сессий
	Sess SessionManager // по сути этот интерфейс нужен только для улучшения масштабируемости
	//  и читаемости кода и соблюдения SOLID
}
type Storage interface {
	Add(*User) error
}
