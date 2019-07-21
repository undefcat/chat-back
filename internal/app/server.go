package app

import (
	"container/list"
	"sync"

	"chat-server/internal/app/payload"
)

type Server struct {
	// 채팅방 리스트
	rooms *list.List

	// 로비에 있는 클라이언트들
	clients *sync.Map

	// 로그인 채널
	Login chan *Client

	// 로그아웃 채널
	Logout chan *Client

	// 채팅방 입장 채널
	Enter chan *Enter

	// 채팅방 생성 채널
	createRoom chan *ChatRoom

	// 채팅방 제거 채널
	destroyRoom chan *ChatRoom
}

// 채팅방 입장
type Enter struct {
	id     int
	client *Client
}

func NewServer() *Server {
	return &Server{
		rooms:      list.New(),
		clients:    &sync.Map{},
		Login:      make(chan *Client),
		Logout:     make(chan *Client),
		Enter:      make(chan *Enter),
		createRoom: make(chan *ChatRoom),
		destroyRoom: make(chan *ChatRoom),
	}
}

func (it *Server) Run() {
	for {
		select {
		case c := <-it.Login:
			it.clients.Store(c, true)
			c.Leave = it.Logout
			c.Send <- it.getRoomList()

		case c := <-it.Logout:
			it.clients.Delete(c)

		case room := <-it.createRoom:
			it.createChatRoom(room)

		case room := <-it.destroyRoom:
			it.destroyChatRoom(room)

		case entered := <-it.Enter:
			it.handleEnter(entered)

		}
	}
}

func (it *Server) getRoomList() *payload.RoomList {
	rooms := make([]interface{}, 0, it.rooms.Len())

	for e := it.rooms.Front(); e != nil; e = e.Next() {
		cr := e.Value.(*ChatRoom)
		rooms = append(rooms, chatRoomToPayLoad(cr))
	}

	return &payload.RoomList{Rooms: rooms}
}

func chatRoomToPayLoad(cr *ChatRoom) *payload.ChatRoom {
	ret := &payload.ChatRoom{
		ID:      float64(cr.ID),
		Title:   cr.Title,
		Total:   float64(cr.Total),
		Current: float64(cr.Current),
	}

	return ret
}

func (it *Server) CreateChatRoom(title string, total int) {
	if total > 16 {
		total = 16
	}

	room := NewChatRoom(title, total, it)
	it.createRoom <-room
}

// 채팅방 고유 id
var id = 0
func (it *Server) createChatRoom(room *ChatRoom) {
	room.ID = id
	id++

	it.rooms.PushBack(room)
	go room.Run()

	it.updateChatRoomStatus()
}

func (it *Server) updateChatRoomStatus() {
	roomList := it.getRoomList()

	it.clients.Range(func(k, v interface{}) bool {
		k.(*Client).Send <- roomList
		return true
	})
}

func (it *Server) destroyChatRoom(room *ChatRoom) {
	for e := it.rooms.Front(); e != nil; e = e.Next() {
		r := e.Value.(*ChatRoom)
		if r != room {
			continue
		}

		it.rooms.Remove(e)
		it.updateChatRoomStatus()
		return
	}
}

func (it *Server) handleEnter(enter *Enter) {
	for e := it.rooms.Front(); e != nil; e = e.Next() {
		room := e.Value.(*ChatRoom)
		if room.ID != enter.id {
			continue
		}

		if room.Current >= room.Total {
			return
		}

		room.Enter <-enter.client
		it.clients.Delete(enter.client)
		it.updateChatRoomStatus()

		return
	}
}