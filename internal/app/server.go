package app

import (
	"container/list"
	"sync"

	"chat-server/internal/app/payload"
)

type Server struct {
	roomsLocker sync.RWMutex

	// 채팅방 리스트
	rooms *list.List

	// 로비에 있는 클라이언트들
	clients sync.Map

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
		rooms:       list.New(),
		Login:       make(chan *Client),
		Logout:      make(chan *Client),
		Enter:       make(chan *Enter),
		createRoom:  make(chan *ChatRoom),
		destroyRoom: make(chan *ChatRoom),
	}
}

func (it *Server) Run() {
	for {
		select {
		case c := <-it.Login:
			it.clients.Store(c, true)
			c.Leave = it.Logout
			c.Send <-it.getRoomList()

		case c := <-it.Logout:
			it.clients.Delete(c)

		case room := <-it.createRoom:
			go it.createChatRoom(room)

		case room := <-it.destroyRoom:
			go it.destroyChatRoom(room)

		case entered := <-it.Enter:
			go it.handleEnter(entered)

		}
	}
}

func (it *Server) getRoomList() payload.RoomList {
	it.roomsLocker.RLock()
	defer it.roomsLocker.RUnlock()

	rooms := make([]interface{}, 0, it.rooms.Len())
	for e := it.rooms.Front(); e != nil; e = e.Next() {
		cr := e.Value.(*ChatRoom)
		rooms = append(rooms, chatRoomToPayLoad(cr))
	}

	return payload.RoomList{
		Type:  payload.TypeRoomList,
		Rooms: rooms,
	}
}

func chatRoomToPayLoad(cr *ChatRoom) payload.ChatRoom {
	ret := payload.ChatRoom{
		ID:        float64(cr.ID),
		Title:     cr.Title,
		Total:     float64(cr.Total()),
		Current:   float64(cr.Current()),
		RoomMaker: float64(cr.RoomMaker),
	}

	return ret
}

func (it *Server) CreateChatRoom(client *Client, title string, total int) {
	if total > 16 {
		total = 16
	}

	room := NewChatRoom(client, it, title, total)
	it.createRoom <-room
}

func (it *Server) createChatRoom(room *ChatRoom) {
	it.roomsLocker.Lock()

	it.rooms.PushBack(room)

	it.roomsLocker.Unlock()

	go room.Run()

	it.updateChatRoomStatus()
}

func (it *Server) updateChatRoomStatus() {
	roomList := it.getRoomList()

	it.clients.Range(func(k, v interface{}) bool {
		k.(*Client).Send <-roomList
		return true
	})
}

func (it *Server) destroyChatRoom(room *ChatRoom) {
	it.roomsLocker.Lock()
	for e := it.rooms.Front(); e != nil; e = e.Next() {
		r := e.Value.(*ChatRoom)
		if r != room {
			continue
		}

		it.rooms.Remove(e)
		break
	}
	it.roomsLocker.Unlock()

	it.updateChatRoomStatus()
}

func (it *Server) handleEnter(enter *Enter) {
	for e := it.rooms.Front(); e != nil; e = e.Next() {
		room := e.Value.(*ChatRoom)
		if room.ID != enter.id {
			continue
		}

		if room.Current() >= room.Total() {
			enter.client.Send <-payload.JoinResponse{
				Type: payload.TypeJoinRoom,
				ID:   -1,
				OK:   false,
			}
			return
		}

		room.Enter <-enter.client
		it.clients.Delete(enter.client)
		it.updateChatRoomStatus()

		return
	}
}
