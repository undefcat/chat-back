package app

import (
	"sync"
	"time"

	"chat-server/internal/app/payload"
)

type ChatRoom struct {
	ID        int
	Title     string
	Total     int
	Current   int
	Server    *Server
	Enter     chan *Client
	Leave     chan *Client
	Clients   *sync.Map
	Broadcast chan interface{}
	Closed    chan struct{}
}

func NewChatRoom(title string, total int, server *Server) *ChatRoom {
	return &ChatRoom{
		Title:     title,
		Total:     total,
		Server:    server,
		Enter:     make(chan *Client),
		Leave:     make(chan *Client),
		Clients:   &sync.Map{},
		Broadcast: make(chan interface{}),
		Closed:    make(chan struct{}),
	}
}

func (it *ChatRoom) Run() {
	go it.closer()

	for {
		select {
		case msg := <-it.Broadcast:
			it.handleBroadcastChatMessage(msg)

		case enter := <-it.Enter:
			it.In(enter)

		case c := <-it.Leave:
			it.Out(c)

		case <-it.Closed:
			return
		}
	}
}

func (it *ChatRoom) In(client *Client) {
	_, exist := it.Clients.LoadOrStore(client, true)
	if exist {
		return
	}

	client.Send <-&payload.JoinResponse{ID: it.ID, OK: true}

	client.Leave = it.Leave
	client.Broadcast = it.Broadcast

	it.Current++
	it.updateRoomStatus()
}

func (it *ChatRoom) Out(client *Client) {
	it.Clients.Delete(client)
	it.Current--

	it.updateRoomStatus()
}

func (it *ChatRoom) updateRoomStatus() {
	it.broadcastUserList()
	it.Server.updateChatRoomStatus()
}

func (it *ChatRoom) handleBroadcastChatMessage(msg interface{}) {
	switch msg.(type) {
	case payload.ChatMessage:
		it.broadcastChatMessage(msg.(payload.ChatMessage))
	}
}

func (it *ChatRoom) broadcastUserList() {
	userList := it.getUserList()

	it.Clients.Range(func(c, _ interface{}) bool {
		client := c.(*Client)
		client.Send <-userList

		return true
	})
}

func (it *ChatRoom) getUserList() *payload.ChatRoomUserList {
	userList := make([]payload.User, 0, it.Current)
	it.Clients.Range(func(c, _ interface{}) bool {
		client := c.(*Client)

		user := payload.User{ID: float64(client.ID), Name: client.Name}
		userList = append(userList, user)
		return true
	})

	return &payload.ChatRoomUserList{UserList: userList}
}

// 채팅 메세지 id
var chatID = 0

func (it *ChatRoom) broadcastChatMessage(msg payload.ChatMessage) {
	msg.ID = float64(chatID)
	chatID++

	it.Clients.Range(func(c, _ interface{}) bool {
		client := c.(*Client)
		client.Send <-&msg

		return true
	})
}

func (it *ChatRoom) closer() {
	for {
		check := time.After(10*time.Second)

		select {
		case <-check:
			if it.Current == 0 {
				it.close()
				return
			}
		}
	}
}

func (it *ChatRoom) close() {
	it.Closed <-struct{}{}
	it.Server.destroyRoom <-it
}