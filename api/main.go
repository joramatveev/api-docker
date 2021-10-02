package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
)

const host string = ""
const port string = "8080"

func getCakeHandler(w http.ResponseWriter, _ *http.Request, u User) {
	_, err := w.Write([]byte(u.FavoriteCake))
	if err != nil {
		return
	}
}

func wrapJWT(
	myJWTService *MyJWTService,
	f func(http.ResponseWriter, *http.Request, *MyJWTService),
) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		f(rw, r, myJWTService)
	}
}

func main() {
	r := mux.NewRouter()

	userService := UserService{
		toasts:     make(chan []byte, 10),
		repository: NewInMemoryUserStorage(),
	}

	myJWTService, err := NewMyJWTService()
	if err != nil {
		panic(err)
	}

	go initPublisher(userService.toasts)
	go initPrometheus()

	r.HandleFunc("/user/me", logRequest(myJWTService.jwtAuth(userService.repository, getCakeHandler))).Methods(http.MethodGet)
	r.HandleFunc("/user/favorite_cake", logRequest(myJWTService.jwtAuth(userService.repository, userService.UpdateCake))).Methods(http.MethodPut)
	r.HandleFunc("/user/email", logRequest(myJWTService.jwtAuth(userService.repository, userService.UpdateEmail))).Methods(http.MethodPut)
	r.HandleFunc("/user/password", logRequest(myJWTService.jwtAuth(userService.repository, userService.UpdatePassword))).Methods(http.MethodPut)

	r.HandleFunc("/user/register", logRequest(userService.Register)).Methods(http.MethodPost)
	r.HandleFunc("/user/jwt", logRequest(wrapJWT(myJWTService, userService.JWT))).Methods(http.MethodPost)

	r.HandleFunc("/admin/ban", logRequest(myJWTService.jwtAuth(userService.repository, userService.BanUser))).Methods(http.MethodPost)
	r.HandleFunc("/admin/unban", logRequest(myJWTService.jwtAuth(userService.repository, userService.UnBanUser))).Methods(http.MethodPost)
	r.HandleFunc("/admin/inspect", logRequest(myJWTService.jwtAuth(userService.repository, userService.History))).Methods(http.MethodGet)

	r.HandleFunc("/admin/promote", logRequest(myJWTService.jwtAuth(userService.repository, userService.Promote))).Methods(http.MethodPost)
	r.HandleFunc("/admin/fire", logRequest(myJWTService.jwtAuth(userService.repository, userService.Fire))).Methods(http.MethodPost)

	srv := http.Server{
		Addr:    host + ":" + port,
		Handler: r,
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			return
		}
	}()

	log.Println("Server started, hit Ctrl+C to stop")
	err = srv.ListenAndServe()
	if err != nil {
		log.Println("Server exited with error:", err)
	}

	log.Println("Good bye :)")
}
