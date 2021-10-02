package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type CakeUpdateParams struct {
	FavoriteCake string `json:"favorite_cake"`
}

type EmailUpdateParams struct {
	Email string `json:"email"`
}

type PasswordUpdateParams struct {
	Password string `json:"password"`
}

func (us *UserService) UpdateCake(w http.ResponseWriter, r *http.Request, u User) {
	params := &CakeUpdateParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("important parameters are missing"), w)
		return
	}

	if err := validateCake(params.FavoriteCake); err != nil {
		handleError(err, w)
		return
	}

	u.FavoriteCake = params.FavoriteCake
	if err := us.repository.Update(u.Email, u); err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err := w.Write([]byte("favorite cake updated"))
	if err != nil {
		return
	}

	us.toasts <- []byte("updated favorite cake: " + u.Email)
}

func (us *UserService) UpdateEmail(w http.ResponseWriter, r *http.Request, u User) {
	params := &EmailUpdateParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("important parameters are missing"), w)
		return
	}

	if err := validateEmail(params.Email); err != nil {
		handleError(err, w)
		return
	}

	if newU, err := us.repository.Delete(u.Email); err != nil {
		handleError(err, w)
		return
	} else {
		newU.Email = params.Email
		if err = us.repository.Add(newU.Email, newU); err != nil {
			handleError(err, w)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	_, err := w.Write([]byte("email updated"))
	if err != nil {
		return
	}

	us.toasts <- []byte("updated email: " + u.Email + " to " + params.Email)
}

func (us *UserService) UpdatePassword(w http.ResponseWriter, r *http.Request, u User) {
	params := &PasswordUpdateParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("important parameters are missing"), w)
		return
	}

	if err := validatePassword(params.Password); err != nil {
		handleError(err, w)
		return
	}

	u.PasswordDigest = string(md5.New().Sum([]byte(params.Password)))
	if err := us.repository.Update(u.Email, u); err != nil {
		handleError(err, w)
		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if err := us.repository.AddToken(token); err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err := w.Write([]byte("password updated"))
	if err != nil {
		return
	}

	us.toasts <- []byte("password updated: " + u.Email)

}
