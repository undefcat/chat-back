# chat-back

[DEMO Page](https://chat.taku.kr)

- [gorilla/websocket](https://github.com/gorilla/websocket)을 이용한 웹소켓 서버

## Server

[Server](https://github.com/undefcat/chat-back/blob/master/internal/app/server.go)는 대기실에 있는 [Client](https://github.com/undefcat/chat-back/blob/master/internal/app/client.go) 및 [ChatRoom](https://github.com/undefcat/chat-back/blob/master/internal/app/chatroom.go)을 관리하는 구조체다.

- `ChatRoom`이 생성되면 리스트에 추가하고, 사라지면 리스트에서 제거한다.

- 클라이언트가 특정 `ChatRoom`에 접속을 요청하면 해당 `ChatRoom`에 입장을 요청한 클라이언트의 정보를 보내준다.

## ChatRoom

[ChatRoom](https://github.com/undefcat/chat-back/blob/master/internal/app/chatroom.go)은 채팅방을 나타내는 구조체다.

- `Server`로부터 `Client`의 입장 요청을 받으면, 채팅방 인원에 따라 입장 여부를 처리한다.

- 채팅방의 정보가 변경되면(인원수) `Server`의 [`updateChatRoomStatue`](https://github.com/undefcat/chat-back/blob/master/internal/app/server.go#L125)를 호출하여 정보가 업데이트 되었다는 것을 알려 대기실에 있는 유저들에게 업데이트된 채팅방의 정보를 전달할 수 있게 한다.

- 채팅방의 메세지를 `Client`들에게 전달하고 귓속말, 입장/퇴장 안내메세지, 강퇴 등도 처리한다.

- 채팅방 방장은 최초 방을 개설한 사람이며, 나갔다 들어와도 방장은 유지된다.

- 채팅방은 생성된 후 10초 간격마다 인원수를 체크하며, 인원수가 0이면 닫힌다.

## Client

[Client](https://github.com/undefcat/chat-back/blob/master/internal/app/client.go)는 클라이언트를 나타내는 구조체다.

- 최초 접속시 `Server`에 로그인한다.

- `ChatRoom`에 입장을 성공하면, `Server`에서는 로그아웃되고 `ChatRoom`에 로그인한다.

- `Server`, `Client`에 로그인할 때마다 각각 `Broadcast`채널을 할당받는다.

- 브라우저로부터 메세지를 받으면, `Broadcast`채널에 메세지를 그대로 전달한다.

- 현재 로그인되어 있는 곳에서 `Send`채널로 메세지를 보내면, 해당 메세지를 브라우저에 전달한다.