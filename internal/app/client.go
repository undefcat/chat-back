package app

import (
	"bytes"
	"encoding/json"
	"log"
	"time"

	"chat-server/internal/app/payload"

	"github.com/gorilla/websocket"
)

// 클라이언트
// Name은 닉네임 설정 요청이 들어오면 처리한다.
type Client struct {
	// 메인 서버
	Server *Server

	// 나가는 채널
	// 서버에 접속하면 서버 로그아웃 채널을 할당해주고
	// 채팅방에 접속하면 채팅방에서 나가는 채널을 할당해준다.
	Leave chan<- *Client

	Conn      *websocket.Conn
	ID        int
	Name      string
	IP        string
	Send      chan interface{}
	Broadcast chan interface{}
}

var clientID = 0

func NewClient(server *Server, conn *websocket.Conn, ip string) *Client {
	c := &Client{
		Server: server,
		Conn:   conn,
		ID:     clientID,
		IP:     ip,
		Send:   make(chan interface{}),
	}

	clientID++

	go c.request()
	go c.listen()

	return c
}

// 클라이언트로부터 데이터를 받아서 처리한다.
func (it *Client) request() {
	defer it.close()

	err := it.Conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	if err != nil {
		log.Println("SetReadDeadline: ", err)
		return
	}

	for {
		_, m, err := it.Conn.ReadMessage()
		if err != nil {
			log.Println("ReadMessage: ", err)
			return
		}

		messageType, message := getMessage(bytes.TrimSpace(m))

		switch messageType {
		case payload.TypeSetName:
			err := it.handleSetName(message)
			if err != nil {
				return
			}

		case payload.TypeCreateRoom:
			err := it.handleCreateRoom(message)
			if err != nil {
				return
			}

		case payload.TypeJoinRoom:
			err := it.handleJoinRoom(message)
			if err != nil {
				return
			}

		case payload.TypeChatMessage:
			err := it.handleBroadcastChatMessage(message)
			if err != nil {
				return
			}

		case payload.TypeLeaveRoom:
			it.handleLeaveRoom()

		}
	}
}

var sep = []byte{'\r', '\n', '\r', '\n'}

func getMessage(m []byte) (string, []byte) {
	split := bytes.Split(m, sep)

	messageType := string(split[0])
	var body []byte

	if len(split) > 1 {
		body = split[1]
	}

	return messageType, body
}

// 서버로부터 데이터를 받아서 클라이언트로 푸쉬해준다.
func (it *Client) listen() {
	defer it.close()

	for {
		select {
		case msg, ok := <-it.Send:
			if !ok {
				return
			}

			switch msg.(type) {
			case *payload.SetNameResponse:
				err := it.handleSetNamePush(msg.(*payload.SetNameResponse))
				if err != nil {
					return
				}

			case *payload.RoomList:
				err := it.handleRoomListPush(msg.(*payload.RoomList))
				if err != nil {
					return
				}

			case *payload.ChatMessage:
				err := it.handleSendChatMessage(msg.(*payload.ChatMessage))
				if err != nil {
					return
				}

			case *payload.NoticeMessage:
				err := it.handleSendNoticeMessage(msg.(*payload.NoticeMessage))
				if err != nil {
					return
				}

			case *payload.JoinResponse:
				err := it.handleJoinRoomPush(msg.(*payload.JoinResponse))
				if err != nil {
					return
				}

			case *payload.ChatRoomUserList:
				err := it.handleUserListPush(msg.(*payload.ChatRoomUserList))
				if err != nil {
					return
				}
			}
		}
	}
}

func (it *Client) handleRoomListPush(msg *payload.RoomList) error {
	msg.Type = payload.TypeRoomList

	err := it.Conn.WriteJSON(msg)
	if err != nil {
		log.Println("handleRoomListPush: ", err)
		return err
	}

	return nil
}

func (it *Client) handleSetName(msg []byte) error {
	var setName payload.SetNameRequest

	err := json.Unmarshal(msg, &setName)
	if err != nil {
		log.Println("handleSetName: ", err)
		return err
	}

	it.Name = setName.Name
	it.Send <-&payload.SetNameResponse{OK: true}

	return nil
}

func (it *Client) handleSetNamePush(msg *payload.SetNameResponse) error {
	msg.Type = payload.TypeSetName

	err := it.Conn.WriteJSON(msg)
	if err != nil {
		log.Println("handleSetNamePush: ", err)
		return err
	}

	return nil
}
func (it *Client) handleCreateRoom(msg []byte) error {
	var room payload.CreateRoomRequest

	err := json.Unmarshal(msg, &room)
	if err != nil {
		log.Println("CreateRoomRequest: ", err)
		return err
	}

	it.Server.CreateChatRoom(room.Title, int(room.Total))

	return nil
}

func (it *Client) handleBroadcastChatMessage(msg []byte) error {
	var chatMessage payload.ChatMessage

	err := json.Unmarshal(msg, &chatMessage)
	if err != nil {
		log.Println("BroadcastChatMessage: ", err)
		return err
	}

	chatMessage.Name = it.Name

	it.Broadcast <-chatMessage
	return nil
}

func (it *Client) handleSendChatMessage(msg *payload.ChatMessage) error {
	msg.Type = payload.TypeChatMessage

	err := it.Conn.WriteJSON(msg)
	if err != nil {
		log.Println("handleSendChatMessage: ", err)
		return err
	}

	return nil
}

func (it *Client) handleSendNoticeMessage(msg *payload.NoticeMessage) error {
	msg.Type = payload.TypeNoticeMessage

	err := it.Conn.WriteJSON(msg)
	if err != nil {
		log.Println("handleSendNoticeMessage: ", err)
		return err
	}

	return nil
}

func (it *Client) handleJoinRoom(msg []byte) error {
	var joinRequest payload.JoinRequest

	err := json.Unmarshal(msg, &joinRequest)
	if err != nil {
		log.Println("handleJoinRoom: ", err)
		return err
	}

	it.Server.Enter <- &Enter{id: int(joinRequest.ID), client: it}
	return nil
}

func (it *Client) handleJoinRoomPush(msg *payload.JoinResponse) error {
	msg.Type = payload.TypeJoinRoom

	err := it.Conn.WriteJSON(msg)
	if err != nil {
		log.Println("handleJoinRoomPush: ", err)
		return err
	}

	return nil
}

func (it *Client) handleUserListPush(msg *payload.ChatRoomUserList) error {
	msg.Type = payload.TypeChatRoomUserList

	err := it.Conn.WriteJSON(msg)
	if err != nil {
		log.Println("handleUserListPush: ", err)
		return err
	}

	return nil
}

func (it *Client) handleLeaveRoom() {
	it.Leave <-it
	it.Server.Login <-it
}

func (it *Client) close() {
	it.Leave <- it
	err := it.Conn.Close()
	if err != nil {
		log.Println("client close: ", err)
	}
}
