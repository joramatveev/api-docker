package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/joramatveev/api-docker/pkg/jwt"
)

const wsHost string = ""
const wsPort string = "8085"

var addr = flag.String("addr", wsHost+":"+wsPort, "http service address")

func main() {
	flag.Parse()
	hub := NewHub()
	go hub.run()
	go hub.receive()

	jwtService, err := jwt.NewJWTService()
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if _, err := jwtService.ParseJWT(token); err != nil {
			w.WriteHeader(401)
			w.Write([]byte("unauthorized"))
			return
		}

		listener(hub, w, r)
	})

	err = http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
