package payload

// 최초 접속후 닉네임 설정
const TypeSetName = "setName"

type SetNameRequest struct {
	Name string `json:"name"`
}

type SetNameResponse struct {
	Type string `json:"type"`
	OK   bool   `json:"ok"`
}

// 채팅방 접속 요청
const TypeJoinRoom = "joinRoom"

type JoinRequest struct {
	ID float64 `json:"id"`
}

type JoinResponse struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
	OK   bool   `json:"ok"`
}

// 채팅방 나가기
const TypeLeaveRoom = "leaveRoom"

type LeaveRoom struct {
	ID float64 `json:"id"`
}

// 채팅방 생성 요청
const TypeCreateRoom = "createRoom"

type CreateRoomRequest struct {
	Title string  `json:"title"`
	Total float64 `json:"total"`
}

// 채팅방 생성 응답
type CreateRoomResponse struct {
	Type string  `json:"type"`
	OK   float64 `json:"ok"`
	ID   float64 `json:"id"`
}

// 채팅방 정보
type ChatRoom struct {
	ID      float64 `json:"id"`
	Title   string  `json:"title"`
	Total   float64 `json:"total"`
	Current float64 `json:"current"`
}

// 채팅방 리스트
// 위의 ChatRoom 값들이 들어있다.
const TypeRoomList = "roomList"

type RoomList struct {
	Type  string        `json:"type"`
	Rooms []interface{} `json:"rooms"`
}

// 채팅방 접속 유저 리스트
const TypeChatRoomUserList = "userList"

type ChatRoomUserList struct {
	Type     string `json:"type"`
	UserList []User `json:"userList"`
}

type User struct {
	ID   float64 `json:"id"`
	Name string  `json:"name"`
}

// 채팅 메세지
const TypeChatMessage = "chatMessage"

type ChatMessage struct {
	Type    string  `json:"type"`
	ID      float64 `json:"id"`
	Name    string  `json:"name"`
	Content string  `json:"content"`
}

const TypeNoticeMessage = "notice"

type NoticeMessage struct {
	Type       string  `json:"type"`
	ID         float64 `json:"id"`
	NoticeType string  `json:"noticeType"`
	Content    string  `json:"content"`
}
