package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
) //проблема на этапе парсинга , нужно парсить id + почту

func (st *StDb) Create(w http.ResponseWriter, user *User) error {
	CookieVal := RandStringRunes(32)
	_, err := st.db.Exec("INSERT INTO Sessions(user_id,cookie) VALUES(?, ?)", user.Id, CookieVal)
	if err != nil {
		log.Printf("Ошибка при создании сессии: %v", err)
		return err
	}
	cookie := &http.Cookie{
		Name:     "session",
		Value:    CookieVal,
		Expires:  time.Now().Add(1 * 24 * time.Hour),
		Path:     "/",
		HttpOnly: true, // Запрет взаимодействий куки с браузером , только бэекнд сервер
		//Secure:   true, // mode HTTPS only
		SameSite: 2, // lax mode
	}
	http.SetCookie(w, cookie)
	return nil
}

func (st *StDb) Check(r *http.Request) (*Session, error) {
	SessionCook, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		log.Println("No cookie")
		return nil, fmt.Errorf("no session cookie")
	}
	var userID int
	err = st.db.QueryRow("SELECT user_id FROM Sessions WHERE cookie = ?", SessionCook.Value).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %v", err)
	}
	var temp User
	err = st.db.QueryRow("SELECT id, username FROM Users WHERE id = ?", userID).Scan(&temp.Id, &temp.Username)
	if err != nil {
		return nil, fmt.Errorf("invalid user session: %v", err)
	}
	_, err = st.db.Exec("UPDATE Sessions SET updated_at = NOW()")
	if err != nil {
		return nil, fmt.Errorf("invalid user session: %v", err)
	}
	return &Session{
		UserID:   temp.Id,
		Username: temp.Username,
	}, nil
}

func (st *StDb) DestroyCurrent(w http.ResponseWriter, r *http.Request) error {
	sess, err := SessionFromContext(r.Context())
	if err == nil {
		_, err = st.db.Exec("DELETE FROM Sessions WHERE id = ?", sess.UserID)
		if err != nil {
			return err
		}
	}
	cookie := http.Cookie{
		Name:    "session",
		Expires: time.Now().AddDate(0, 0, -1),
		Path:    "/",
	}
	http.SetCookie(w, &cookie)
	return nil
}

func (st *StDb) DestroyAll(user *User) error { //w http.ResponseWriter , для чего он нужен был в примере хз
	result, err := st.db.Exec("DELETE FROM Sessions WHERE id = ?", user.Id)
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	log.Println("destroyed sessions", affected, "for user", user.Id)
	return nil
}
