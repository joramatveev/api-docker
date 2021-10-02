package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

type BanUserParams struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

type UnBanUserParams struct {
	Email string `json:"email"`
}

type PromoteParams struct {
	Email string `json:"email"`
}

type FireParams struct {
	Email string `json:"email"`
}

func (us *UserService) BanUser(w http.ResponseWriter, r *http.Request, u User) {
	params := &BanUserParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("important parameters are missing"), w)
		return
	}

	user, err := us.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	if len(u.Role) <= len(user.Role) {
		handleError(errors.New("no privileges, access denied"), w)
		return
	}

	err = us.repository.Ban(params.Email, u.Email, params.Reason)
	if err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte("user \"" + params.Email + "\" is banned with reason\"" + params.Reason + "\" by \"" + u.Email + "\""))
	if err != nil {
		return
	}

	us.toasts <- []byte("IN BAN: " + params.Email)
}

func (us *UserService) UnBanUser(w http.ResponseWriter, r *http.Request, u User) {
	params := &UnBanUserParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("important parameters are missing"), w)
		return
	}

	user, err := us.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	if len(u.Role) <= len(user.Role) {
		handleError(errors.New("no privileges, access denied"), w)
		return
	}

	err = us.repository.UnBan(params.Email, u.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte("user \"" + params.Email + "\" is UnBanned by \"" + u.Email + "\""))
	if err != nil {
		return
	}

	us.toasts <- []byte("OUT BAN: " + params.Email)
}

func (us *UserService) History(w http.ResponseWriter, r *http.Request, u User) {
	email := r.URL.Query().Get("email")
	if len(email) == 0 {
		handleError(errors.New("email is empty"), w)
		return
	}

	user, err := us.repository.Get(email)
	if err != nil {
		handleError(err, w)
		return
	}

	if len(u.Role) <= len(user.Role) {
		handleError(errors.New("no privileges, access denied"), w)
		return
	}

	history, err := us.repository.BanHistory(email)
	if err != nil {
		handleError(err, w)
		return
	}

	body, err := json.Marshal(history)
	if err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(body)
	if err != nil {
		return
	}
}

func (us *UserService) Fire(w http.ResponseWriter, r *http.Request, u User) {
	params := &FireParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("important parameters are missing"), w)
		return
	}

	user, err := us.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	if u.Email == user.Email {
		handleError(errors.New("you cannot change the role to yourself"), w)
		return
	}

	if "superadmin" != string(u.Role) {
		handleError(errors.New("no privileges, access denied"), w)
		return
	}

	if "admin" != string(user.Role) {
		handleError(errors.New("user does not have admin privileges"), w)
		return
	}

	user.Role = "user"
	if err := us.repository.Update(params.Email, user); err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte("user \"" + u.Email + "\" has revoked the privileges of user \"" + user.Email + "\""))
	if err != nil {
		return
	}
}

func (us *UserService) Promote(w http.ResponseWriter, r *http.Request, u User) {
	params := &PromoteParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		handleError(errors.New("important parameters are missing"), w)
		return
	}

	user, err := us.repository.Get(params.Email)
	if err != nil {
		handleError(err, w)
		return
	}

	if u.Email == user.Email {
		handleError(errors.New("you cannot change the role to yourself"), w)
		return
	}

	if "superadmin" != u.Role {
		handleError(errors.New("no privileges, access denied"), w)
		return
	}

	if "user" != string(user.Role) {
		handleError(errors.New("user does not have user privileges"), w)
		return
	}

	user.Role = "admin"
	if err := us.repository.Update(params.Email, user); err != nil {
		handleError(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte("user \"" + u.Email + "\" changed the privileges of user \"" + user.Email + "\" to \"admin\""))
	if err != nil {
		return
	}
}
