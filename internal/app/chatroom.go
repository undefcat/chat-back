package app

import (
	"fmt"
	"sync"
	"time"

	"chat-server/internal/app/payload"
)

type ChatRoom struct {
	// total, current 동기화를 위한
	// 임베디드 RWMutex
	sync.RWMutex

	// 채팅방 고유 ID
	ID int

	// 채팅방 제목
	Title string

	// 총 인원
	// getter로 값을 가져와야 한다.
	total int

	// 현재 인원
	// IncCurrent, DecCurrent로 값을 설정해야 한다.
	current int

	// 방장의 ID
	// 채팅방을 나갔다 들어와도 방장은 유지된다.
	RoomMaker int

	// 서버
	Server *Server

	// 입장 채널
	Enter chan *Client

	// 퇴장 채널
	Leave chan *Client

	// 채팅방에 있는 클라이언트들
	Clients *sync.Map

	// 브로드캐스트 채널
	Broadcast chan interface{}

	// 채팅방 닫는 채널
	Closed chan struct{}
}

var chatRoomIDGenerator chan int

func init() {
	chatRoomIDGenerator = make(chan int)
	id := 0

	go func() {
		for {
			chatRoomIDGenerator <-id
			id++
		}
	}()
}

func NewChatRoom(c *Client, s *Server, title string, total int) *ChatRoom {
	return &ChatRoom{
		ID:        <-chatRoomIDGenerator,
		Title:     title,
		total:     total,
		RoomMaker: c.ID,
		Server:    s,
		Enter:     make(chan *Client),
		Leave:     make(chan *Client),
		Clients:   &sync.Map{},
		Broadcast: make(chan interface{}),
		Closed:    make(chan struct{}),
	}
}

func (it *ChatRoom) Total() int {
	it.RLock()
	defer it.RUnlock()
	return it.total
}

func (it *ChatRoom) Current() int {
	it.RLock()
	defer it.RUnlock()
	return it.current
}

// it.current++
func (it *ChatRoom) IncCurrent() {
	it.Lock()
	defer it.Unlock()
	it.current++
}

// it.current--
func (it *ChatRoom) DecCurrent() {
	it.Lock()
	defer it.Unlock()
	it.current--
}

func (it *ChatRoom) Run() {
	go it.closer()

	for {
		select {
		case msg := <-it.Broadcast:
			go it.handleBroadcast(msg)

		case enter := <-it.Enter:
			go it.In(enter)

		case c := <-it.Leave:
			go it.Out(c)

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

	client.Send <-payload.JoinResponse{
		Type: payload.TypeJoinRoom,
		ID:   it.ID,
		OK:   true,
	}

	client.Leave = it.Leave
	client.Broadcast = it.Broadcast

	it.Broadcast <-payload.NoticeMessage{
		NoticeType: "enter",
		Content:    fmt.Sprintf("%s 님이 입장하셨습니다.", client.Name),
	}

	it.IncCurrent()
	it.updateRoomStatus()
}

func (it *ChatRoom) Out(client *Client) {
	it.Clients.Delete(client)
	if it.Current() > 0 {
		it.DecCurrent()
	}

	it.Broadcast <-payload.NoticeMessage{
		NoticeType: "leave",
		Content:    fmt.Sprintf("%s 님이 나가셨습니다.", client.Name),
	}

	it.updateRoomStatus()
}

func (it *ChatRoom) updateRoomStatus() {
	it.broadcastUserList()
	it.Server.updateChatRoomStatus()
}

func (it *ChatRoom) broadcastUserList() {
	userList := it.getUserList()

	it.Clients.Range(func(c, _ interface{}) bool {
		client := c.(*Client)
		client.Send <-userList

		return true
	})
}

func (it *ChatRoom) getUserList() payload.ChatRoomUserList {
	userList := make([]payload.User, 0, it.Current())
	it.Clients.Range(func(c, _ interface{}) bool {
		client := c.(*Client)

		user := payload.User{ID: float64(client.ID), Name: client.Name}
		userList = append(userList, user)
		return true
	})

	return payload.ChatRoomUserList{
		Type: payload.TypeChatRoomUserList,
		UserList: userList,
	}
}

func (it *ChatRoom) handleBroadcast(msg interface{}) {
	switch msg.(type) {
	case payload.ChatMessage:
		it.broadcastChatMessage(msg.(payload.ChatMessage))

	case payload.NoticeMessage:
		it.broadcastNoticeMessage(msg.(payload.NoticeMessage))

	case payload.WhisperMessage:
		it.broadcastWhisperMessage(msg.(payload.WhisperMessage))

	case payload.BanUser:
		it.banUser(msg.(payload.BanUser))
	}
}

var chatIDGenerator chan float64

func init() {
	chatIDGenerator = make(chan float64)

	id := 0.0
	go func() {
		for {
			chatIDGenerator <-id
			id++
		}
	}()
}

func (it *ChatRoom) broadcastChatMessage(msg payload.ChatMessage) {
	msg.Type = payload.TypeChatMessage
	msg.ID = <-chatIDGenerator

	it.Clients.Range(func(c, _ interface{}) bool {
		client := c.(*Client)
		client.Send <-msg

		return true
	})
}

func (it *ChatRoom) broadcastNoticeMessage(msg payload.NoticeMessage) {
	msg.Type = payload.TypeNoticeMessage
	msg.ID = <-chatIDGenerator

	it.Clients.Range(func(c, _ interface{}) bool {
		client := c.(*Client)
		client.Send <-msg

		return true
	})
}

func (it *ChatRoom) broadcastWhisperMessage(msg payload.WhisperMessage) {
	msg.Type = payload.TypeWhisperMessage
	msg.ID = <-chatIDGenerator

	it.Clients.Range(func(c, _ interface{}) bool {
		client := c.(*Client)
		if client.ID == int(msg.ToID) || client.ID == int(msg.FromID) {
			client.Send <-msg
		}

		return true
	})
}

func (it *ChatRoom) banUser(msg payload.BanUser) {
	if it.RoomMaker != int(msg.ID) {
		return
	}

	msg.Type = payload.TypeBanUser

	it.Clients.Range(func(c, _ interface{}) bool {
		client := c.(*Client)
		if client.ID == int(msg.BanID) {
			client.Send <-msg

			return false
		}

		return true
	})
}

func (it *ChatRoom) closer() {
	for {
		select {
		case <-time.After(10 * time.Second):
			if it.Current() == 0 {
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
