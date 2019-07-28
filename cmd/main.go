package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"chat-server/internal/app"

	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

func init() {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
}

var s *app.Server

var port string

func init() {
	flag.StringVar(&port, "port", "8000", "port로 서버를 실행한다.")
}

func main() {
	flag.Parse()

	s = app.NewServer()
	go s.Run()

	logFile, err := os.Create("/app/log/chat-server.log")
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(logFile)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(s, w, r)
	})

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalln("ListenAndServe: ", err)
	}
}

func serveWS(s *app.Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade failed: ", err)
		return
	}

	c := app.NewClient(s, conn, r.RemoteAddr)
	s.Login <-c
}
