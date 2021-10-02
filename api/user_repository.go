package main

import (
	"crypto/md5"
	"errors"
	"os"
	"sync"
)

type InMemoryUserStorage struct {
	lock       sync.RWMutex
	storage    map[string]User
	invTokenDB map[string]struct{}
	banHistory map[string][]Ban
}

func (ur *InMemoryUserStorage) Fire(string) error {
	panic("implement me")
}

func (ur *InMemoryUserStorage) Promote(string) error {
	panic("implement me")
}

func NewInMemoryUserStorage() *InMemoryUserStorage {
	ur := InMemoryUserStorage{
		lock:       sync.RWMutex{},
		storage:    make(map[string]User),
		invTokenDB: make(map[string]struct{}),
		banHistory: make(map[string][]Ban),
	}

	suLogin := os.Getenv("CAKE_ADMIN_EMAIL")
	suPassword := os.Getenv("CAKE_ADMIN_PASSWORD")

	_ = ur.Add(suLogin, User{
		Email:          suLogin,
		PasswordDigest: string(md5.New().Sum([]byte(suPassword))),
		Role:           "superadmin",
		FavoriteCake:   "supercake",
	})

	return &ur
}

func (ur *InMemoryUserStorage) Add(login string, u User) error {
	if _, ok := ur.storage[login]; ok {
		return errors.New("this email address is already exists in the database")
	}

	ur.lock.Lock()
	defer ur.lock.Unlock()

	ur.storage[login] = u
	return nil
}

func (ur *InMemoryUserStorage) Get(login string) (User, error) {
	u, ok := ur.storage[login]

	if !ok {
		return User{}, errors.New("user not found")
	} else {
		return u, nil
	}
}

func (ur *InMemoryUserStorage) Update(login string, u User) error {
	if _, ok := ur.storage[login]; !ok {
		return errors.New("user not found")
	}

	ur.lock.Lock()
	defer ur.lock.Unlock()

	ur.storage[login] = u
	return nil
}

func (ur *InMemoryUserStorage) Delete(login string) (User, error) {
	u, ok := ur.storage[login]

	ur.lock.Lock()
	delete(ur.storage, login)
	ur.lock.Unlock()

	if !ok {
		return User{}, errors.New("there is no such user to delete")
	} else {
		return u, nil
	}
}

func (ur *InMemoryUserStorage) CheckNotInDB(jwtToken string) error {
	if _, ok := ur.invTokenDB[jwtToken]; ok {
		return errors.New("token is banned")
	}
	return nil
}

func (ur *InMemoryUserStorage) AddToken(jwtToken string) error {
	if err := ur.CheckNotInDB(jwtToken); err != nil {
		return errors.New("token is already banned")
	}

	ur.lock.Lock()
	defer ur.lock.Unlock()

	ur.invTokenDB[jwtToken] = struct{}{}
	return nil
}
