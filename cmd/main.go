package main

import (
	"chat-server/internal/app"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
)

var upgrader websocket.Upgrader

func init() {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
}

var s *app.Server

func main() {
	s = app.NewServer()
	go s.Run()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Getwd: ", err)
	}

	fs := http.FileServer(http.Dir(wd+"/dist/"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(s, w, r)
	})

	err = http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatalln("ListenAndServe: ", err)
	}
}

var sem = make(chan struct{}, 1)

func serveWS(s *app.Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade failed: ", err)
		return
	}

	// 클라이언트마다 id값은 고유하게 할당되어야 하므로
	// 크기 1짜리 semaphore 사용
	sem <-struct{}{}

	c := app.NewClient(s, conn, r.RemoteAddr)
	s.Login <-c

	<-sem
}
