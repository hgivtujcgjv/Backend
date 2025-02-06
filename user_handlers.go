package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	//"errors"
	"crypto/subtle"
	"io"

	"golang.org/x/crypto/argon2"
)

// np auths тоже потребуется так как я никак не могу с помощью одного мидалываря обработать страрницы без сессии
// надо сдлеать большую мапу чтобы засунуть туда все что не требует явной куки сессси
func (uh *UserHandler) SwitchUserMethods(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		uh.RegistrateUser(w, r)
	case http.MethodGet:
		uh.GetUserByToken(w, r)
	case http.MethodPut:
		uh.UpdateUser(w, r)
	default:
		http.Error(w, "Error unkonown Method type", http.StatusBadRequest)
	}
}

// для юзеров данный метод требует вегда проверки токенов
// states 0 - без требований авторизации , 1 - требования авторизации , 2 - полученный метож не существует стоит вернуть 404
func SwitchUserMethodsAuthRequir(r *http.Request) int {
	switch r.Method {
	case http.MethodPost:
		return 0
	case http.MethodGet:
		return 1
	case http.MethodPut:
		return 1
	default:
		return 2
	}
}

func (uh *UserHandler) GetUserByToken(w http.ResponseWriter, r *http.Request) {
	SessionCook, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		log.Println("No cookie")
		http.Error(w, "Error while parsing request", http.StatusUnauthorized)
		return
	}
	var userID int
	err = uh.Bd.QueryRow("SELECT user_id FROM Sessions WHERE cookie = ?", SessionCook.Value).Scan(&userID)
	if err != nil {
		http.Error(w, "Error while parsing request", http.StatusNotAcceptable)
		return
	}
	var temp UserResponse
	temp.Token = SessionCook.Value
	var bio, image sql.NullString
	err = uh.Bd.QueryRow("SELECT email, username, bio, image FROM Users WHERE id = ?", userID).Scan(&temp.Email, &temp.Username, &bio, &image)
	if err != nil {
		temperr := fmt.Sprintf("Error by cookie parsing %v", err)
		http.Error(w, temperr, http.StatusNotAcceptable)
		return
	}
	if bio.Valid {
		temp.Bio = bio.String
	}
	if image.Valid {
		temp.Image = image.String
	}
	writeJSONResponse(w, temp)
}

// первее проверить методы которые не требуют аунтентификации
func (uh *UserHandler) RegistrateUser(w http.ResponseWriter, r *http.Request) {
	var us User
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Error while parsing request 1", http.StatusInternalServerError)
		return
	}
	err1 := json.Unmarshal(body, &us)
	if err1 != nil {
		http.Error(w, "Error while parsing request", http.StatusInternalServerError)
		return
	}
	if us.Email == "" || us.Password == "" || us.Username == "" {
		http.Error(w, "Mismatched input fields", http.StatusBadRequest)
		return
	}
	salt := RandStringRunes(8)
	us.Password = uh.HashPassword(us.Password, salt)
	Result, Err := uh.Bd.Exec("INSERT INTO Users(email, password, username, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())", us.Email, us.Password, us.Username)
	if Err != nil {
		http.Error(w, fmt.Sprintf("Login or email already exist, try another"), http.StatusBadRequest)
		return
	}
	affected, _ := Result.RowsAffected()
	if affected == 0 {
		http.Error(w, "Looks like user exists", http.StatusBadRequest)
		return
	}
	LastInsertId, Err := Result.LastInsertId()
	if Err != nil {
		http.Error(w, "Internal server error , inserting id problem", http.StatusInternalServerError)
		return
	}
	if LastInsertId == 0 {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	us.Id = int(LastInsertId)
	uh.Sess.Create(w, &us)
	w.WriteHeader(http.StatusCreated)
	writeJSONResponse(w, us.Username)
}

func (uh *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var us User
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Error while parsing request 1", http.StatusInternalServerError)
		return
	}
	err1 := json.Unmarshal(body, &us)
	if err1 != nil {
		http.Error(w, "Error while parsing request", http.StatusInternalServerError)
		return
	}
	MassiveFields := []string{}
	var QueryParametrs []interface{}
	if us.Email != "" {
		MassiveFields = append(MassiveFields, "email =  ?")
		QueryParametrs = append(QueryParametrs, us.Email)
	}
	if us.Password != "" {
		MassiveFields = append(MassiveFields, "password =  ?")
		salt := RandStringRunes(8)
		QueryParametrs = append(QueryParametrs, uh.HashPassword(us.Password, salt))
	}
	if us.Username != "" {
		MassiveFields = append(MassiveFields, "username =  ?")
		QueryParametrs = append(QueryParametrs, us.Username)
	}
	if us.Bio != "" {
		MassiveFields = append(MassiveFields, "bio =  ?")
		QueryParametrs = append(QueryParametrs, us.Bio)
	}
	if us.Image != "" {
		MassiveFields = append(MassiveFields, "image =  ?")
		QueryParametrs = append(QueryParametrs, us.Image)
	}
	if len(MassiveFields) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}
	fmt.Println(fmt.Sprintf("UPDATE Users Set %s", strings.Join(MassiveFields, ", ")))
	QueryString := fmt.Sprintf("UPDATE Users Set %s", strings.Join(MassiveFields, ", "))
	Result, Err := uh.Bd.Exec(QueryString, QueryParametrs...)
	if Err != nil {
		http.Error(w, fmt.Sprintf("Internal while pushing value: %v", Err), http.StatusInternalServerError)
		return
	}
	affected, _ := Result.RowsAffected()
	if affected == 0 {
		http.Error(w, "Looks like user not exists", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "User updated successfully")
}

func (uh *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var us User
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error while parsing request 1", http.StatusInternalServerError)
		return
	}
	err1 := json.Unmarshal(body, &us)
	if err1 != nil {
		http.Error(w, "Error while parsing request", http.StatusInternalServerError)
		return
	}
	Result, Err := uh.Bd.Exec("DELETE FROM Users WHERE email = ?", us.Email)
	if Err != nil {
		http.Error(w, fmt.Sprintf("Internal while pushing value: %v", Err), http.StatusInternalServerError)
		return
	}
	affected, _ := Result.RowsAffected()
	if affected == 0 {
		http.Error(w, "Looks like user exists", http.StatusBadRequest)
		return
	}
	LastInsertId, Err := Result.LastInsertId()
	if Err != nil {
		http.Error(w, "Internal server error , inserting id problem", http.StatusInternalServerError)
		return
	}
	if LastInsertId == 0 {
		http.Error(w, "Internal server error, id = 0 ", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "User was deleted")
}

func (uh *UserHandler) HashPassword(TextPassword, salt string) string {
	HashedPassword := argon2.IDKey([]byte(TextPassword), []byte(salt), 1, 64*1024, 4, 32)
	return fmt.Sprintf("%s%s", salt, base64.StdEncoding.EncodeToString(HashedPassword))
}

func (uh *UserHandler) CheckValidPass(password string, row *sql.Row) (*User, error) {
	var (
		dbPass   sql.NullString
		username sql.NullString
		id       sql.NullString
		user     *User
	)
	err := row.Scan(&id, &username, &dbPass)
	if err == sql.ErrNoRows {
		return nil, errors.New("no user record found")
	} else if err != nil {
		return nil, err
	}
	if id.Valid {
		userId, err := strconv.Atoi(id.String)
		if err != nil {
			return nil, errors.New("invalid user ID format")
		}
		user = &User{Id: userId}
	}
	if username.Valid {
		user.Username = username.String
	}
	if dbPass.Valid {
		user.Password = dbPass.String
	}
	if len(user.Password) <= 8 {
		return nil, errors.New("invalid stored password format")
	}
	salt := user.Password[:8]
	//storedHash := dbPass[8:]
	computedHash := uh.HashPassword(password, salt)
	//computedHashBase64 := computedHash[8:]
	if subtle.ConstantTimeCompare([]byte(computedHash), []byte(user.Password)) != 1 {
		return nil, errors.New("wrong password")
	}
	return user, nil
}

func (uh *UserHandler) checkPasswordByUserID(uid uint32, pass string) (*User, error) {
	row := uh.Bd.QueryRow("SELECT id, username, password FROM Users WHERE id = ?", uid)
	return uh.CheckValidPass(pass, row)
}

func (uh *UserHandler) checkPasswordByUsername(username, pass string) (*User, error) {
	row := uh.Bd.QueryRow("SELECT id, username, email, password FROM Users WHERE username = ?", username)
	return uh.CheckValidPass(pass, row)
}

func (uh *UserHandler) checkPasswordByEmail(email, pass string) (*User, error) {
	row := uh.Bd.QueryRow("SELECT id, username, password FROM Users WHERE email = ?", email)
	return uh.CheckValidPass(pass, row)
}

var (
	errBadPass = errors.New("No user record found")
)

func (uh *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		response := map[string]interface{}{
			"errors": map[string][]string{
				"body": {"Only POST method is allowed"},
			},
		}
		writeJSONResponse(w, response)
		return
	}
	Body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Error while parsing request 1", http.StatusInternalServerError)
		return
	}
	var TempUser User
	err1 := json.Unmarshal(Body, &TempUser)
	if err1 != nil {
		http.Error(w, "Error while parsing request", http.StatusInternalServerError)
		return
	}
	user, err := uh.checkPasswordByEmail(TempUser.Email, TempUser.Password)
	switch err {
	case nil:
		// значит все в порядке
	case errBadPass:
		http.Error(w, "Bad password", http.StatusBadRequest)
	default:
		http.Error(w, "Bad pass", http.StatusBadRequest)
	}
	if err != nil {
		return
	}
	uh.Sess.Create(w, user)
	http.Redirect(w, r, "/users", http.StatusFound)
}

func writeJSONResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, `{"errors":{"body":["Failed to encode response"]}}`, http.StatusInternalServerError)
	}
}

func (uh *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	uh.Sess.DestroyCurrent(w, r)
	http.Redirect(w, r, "/user/login", http.StatusFound)
}
