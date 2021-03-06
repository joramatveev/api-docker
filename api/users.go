package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
)

type User struct {
	Email          string
	PasswordDigest string
	Role           string
	FavoriteCake   string
}

type UserRepository interface {
	Add(string, User) error
	Get(string) (User, error)
	Update(string, User) error
	Delete(string) (User, error)

	CheckNotInDB(string) error
	AddToken(string) error

	IsBanned(string) error
	BanHistory(string) ([]Ban, error)
	Ban(string, string, string) error
	UnBan(string, string) error

	Fire(string) error
	Promote(string) error
}

type UserService struct {
	repository UserRepository
	toasts     chan []byte
	reg        chan bool
	cake       chan bool
}

type UserRegisterParams struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	FavoriteCake string `json:"favorite_cake"`
}

func validateEmail(email string) error {
	if match, _ := regexp.MatchString("^[a-z0-9._%+\\-]+@[a-z0-9.\\-]+\\.[a-z]{2,4}$", email); !match {
		return errors.New("email is not valid")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password length must be at least 8 symbols ")
	}
	return nil
}

func validateCake(cake string) error {
	if cake == "" {
		return errors.New("favorite cake should not be empty")
	}

	if match, _ := regexp.MatchString("^[a-zA-Z]+$", cake); !match {
		return errors.New("favorite cake should have only alphabetic characters")
	}

	return nil
}

func validateRegisterParams(p *UserRegisterParams) error {
	if err := validateEmail(p.Email); err != nil {
		return err
	}

	if err := validatePassword(p.Password); err != nil {
		return err
	}

	if err := validateCake(p.FavoriteCake); err != nil {
		return err
	}

	return nil
}

func (us *UserService) Register(w http.ResponseWriter, r *http.Request) {
	params := &UserRegisterParams{}

	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("important parameters are missing"), w)
		return
	}

	if err := validateRegisterParams(params); err != nil {
		handleError(err, w)
		return
	}

	passwordDigest := md5.New().Sum([]byte(params.Password))
	newUser := User{
		Email:          params.Email,
		PasswordDigest: string(passwordDigest),
		FavoriteCake:   params.FavoriteCake,
		Role:           "user",
	}

	if err := us.repository.Add(params.Email, newUser); err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err := w.Write([]byte("registered"))
	if err != nil {
		return
	}

	us.toasts <- []byte("registered: " + params.Email)
}

func handleError(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	_, err = w.Write([]byte(err.Error()))
	if err != nil {
		return
	}
}
