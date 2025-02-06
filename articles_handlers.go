package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

//	по сути я писал ьэкенд для кондуит и вначале посматривал на тесты с курса степик
//
// но оказалось что сваггер схема и то что написанно в тестах совсем различно
// поэтому у меня получился монстр который не подошел ни туда , ни сюда
// но всю логику я протестировал в ручную
// этот свитч тому пример , он обрабатывает распределение методов с conduit, но потом писать рутинный код стало скучно
// и я решил не допиливать обработку всех методов и перешел к новому проэкту
func SwitchArticlesMethodsAuthRequir(r *http.Request) int {
	switch r.Method {
	case http.MethodPost:
		return 0
	case http.MethodGet:
		str := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(str) > 1 {
			if str[1] == "feed" { // я не стал обрабатывать данный метод
				return 1
			} else {
				return 0 // /articles/{slug}
			}
		} else {
			return 0 // articles случай
		}
	case http.MethodPut:
		return 2
	case http.MethodDelete:
		return 2
	default:
		return 2
	}
}

func (uh *UserHandler) SwitchArticlesMethods(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		uh.CreateArticle(w, r)
	case http.MethodGet:
		uh.GetArticleMassive(w, r)
	case http.MethodPut:
		//uh.UpdateUser(w, r)
	default:
		http.Error(w, "Error unkonown Method type", http.StatusBadRequest)
	}
}

func (uh *UserHandler) CreateArticle(w http.ResponseWriter, r *http.Request) {
	var art Article
	Body, Err := io.ReadAll(r.Body)
	if Err != nil {
		http.Error(w, "Error body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	ErrParse := json.Unmarshal(Body, &art)
	if ErrParse != nil {
		http.Error(w, "Error =(", http.StatusInternalServerError)
		return
	}
	SessionCook, cerr := r.Cookie("session")
	if cerr != nil {
		http.Error(w, "Cookie error ", http.StatusBadRequest)
		return
	}
	var userID int
	ErrPars := uh.Bd.QueryRow("SELECT user_id FROM Sessions WHERE cookie = ?", SessionCook.Value).Scan(&userID)
	if ErrPars != nil {
		http.Error(w, "Cookie Id problem", http.StatusBadRequest)
		return
	}
	var bio, image sql.NullString

	ErrPars2 := uh.Bd.QueryRow("SELECT email, username, bio, image FROM Users WHERE id = ?", userID).
		Scan(&art.Author.Email, &art.Author.Username, &bio, &image)

	if ErrPars2 != nil {
		temp := fmt.Sprintf("%#v", ErrPars2)
		http.Error(w, temp, http.StatusBadRequest)
		return
	}
	if bio.Valid {
		art.Author.Bio = bio.String
	} else {
		art.Author.Bio = ""
	}
	if image.Valid {
		art.Author.Image = image.String
	} else {
		art.Author.Image = ""
	}
	var articleCount int
	ErrCount := uh.Bd.QueryRow("SELECT COUNT(*) FROM Article").Scan(&articleCount)
	if ErrCount != nil {
		http.Error(w, "Error counting articles", http.StatusInternalServerError)
		return
	}
	art.Slug = fmt.Sprintf("slug-%d", articleCount+1)
	tagsJSON, err := json.Marshal(art.TagList)
	if err != nil {
		http.Error(w, "Error converting tags to JSON", http.StatusInternalServerError)
		return
	}
	Result, InErr := uh.Bd.Exec("INSERT INTO Article(userId, slug, body, description, tagList, title, createdAt, updatedAt) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())", userID, art.Slug, art.Body, art.Description, string(tagsJSON), art.Title)
	if InErr != nil {
		temp := fmt.Sprintf("%#v", InErr)
		http.Error(w, temp, http.StatusInternalServerError)
		return
	}
	rows, err := Result.RowsAffected()
	if err != nil {
		http.Error(w, "We have some problems with header", http.StatusInternalServerError)
		return
	}
	if rows == 0 {
		http.Error(w, "0 rows affected", http.StatusInternalServerError)
	}
	writeJSONResponse(w, art)
}

func (uh *UserHandler) GetArticleMassive(w http.ResponseWriter, r *http.Request) {
	Count := r.URL.Query().Get("count")
	TempCount, ConvErr := strconv.Atoi(Count)
	if ConvErr != nil || TempCount <= 0 {
		TempCount = 20
	}
	QueryRows, err := uh.Bd.Query("SELECT slug FROM Article ORDER BY userId LIMIT ? OFFSET 0", TempCount)
	if err != nil {
		temp := fmt.Sprintf("%#v", err)
		http.Error(w, temp, http.StatusInternalServerError)
		return
	}
	defer QueryRows.Close()
	var slugs []string
	for QueryRows.Next() {
		var TempSlug string
		if err := QueryRows.Scan(&TempSlug); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		slugs = append(slugs, TempSlug)
	}
	if err := QueryRows.Err(); err != nil {
		http.Error(w, "Error iterating rows", http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, map[string]interface{}{
		"slugs": slugs,
	})
}
