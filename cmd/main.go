package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"chat-server/internal/app"
	"chat-server/internal/app/payload"

	"github.com/gorilla/websocket"
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

	go command()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(s, w, r)
	})

	err := http.ListenAndServe(":8000", nil)
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

func command() {
	sc := bufio.NewScanner(os.Stdin)

	for sc.Scan() {
		t := sc.Text()

		split := strings.Fields(t)

		cmd := split[0]
		split = split[1:]

		switch cmd {
		case payload.TypeCreateRoom:
			title := split[0]
			total, _ := strconv.Atoi(split[1])

			s.CreateChatRoom(title, total)
		}
	}
}