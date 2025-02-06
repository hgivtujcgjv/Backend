package main

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
)

type ctxKey int

const sessionKey ctxKey = 1

var (
	NoAuthPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^/users/login$`),
		regexp.MustCompile(`^/users/reg$`),
		regexp.MustCompile(`^/$`),
		regexp.MustCompile(`^/articles$`),
	}
)

// Проверяет, подпадает ли путь под список исключений
func isNoAuthPath(path string) bool {
	for _, pattern := range NoAuthPatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

func SessionFromContext(ctx context.Context) (*Session, error) {
	sess, ok := ctx.Value(sessionKey).(*Session)
	if !ok {
		return nil, errors.New("No session found")
	}
	return sess, nil
}

func AuthMiddleware(sm SessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isNoAuthPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		TypePart := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		var AuthType int
		if len(TypePart) > 0 {
			switch TypePart[0] {
			case "user":
				AuthType = SwitchUserMethodsAuthRequir(r)
			case "articles":
				AuthType = SwitchArticlesMethodsAuthRequir(r)
			case "profiles":
				// AuthType = SwitchProfilesMethodsAuthRequir(r)
			}
		}
		switch AuthType {
		case 0:
			next.ServeHTTP(w, r)
			return
		case 1:
			sess, err := sm.Check(r)
			if err != nil {
				http.Error(w, "No auth", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), sessionKey, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		case 2:
			http.Error(w, "Undefined", http.StatusBadRequest)
			return
		}
	})
}
