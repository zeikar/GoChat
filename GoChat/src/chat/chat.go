// 메인 패키지
package main

// 임포트 설정
import(
	"container/list"
	"log"
	"net/http"
	"time"
	"fmt"

	"github.com/googollee/go-socket.io" // socket.io 패키지 사용
)

// 전역 변수
var (
	subscribe = make(chan (chan<- Subscription), 10)// 구독 채널
	unsubscribe = make(chan (<-chan Event), 10)		// 구독 해지 채널
	publish = make(chan Event, 10)					// 이벤트 발행 채널

	register = make(chan User, 10)				// 사용자 가입 채널
	login = make(chan User, 10)					// 사용자 로그인 채널
	loginCheck = make(chan LoginCheckRequest, 10)// 로그인 되어있는지 확인 채널
)

// 로그인한 유저 정보 요청
type LoginCheckRequest struct {
	UserId string			// 유저 닉네임
	Check chan<- string		// 확인된 문자열을 받을 채널
}

// 사용자 정보 구조체 정의
type User struct {
	UserId string			// 유저의 세션 아이디
	UserName string			// 유저 닉네임
	UserChan chan Event		// 유저가 데이터 받을 수 있는 채널
}

// 채팅 이벤트 구조체 정의
type Event struct {
	EvtType string	// 이벤트 타입 (join, leave, message)
	UserId string	// 사용자 이름
	Timestamp int	// 시간 값
	Text string		// 메시지 텍스트
}

// 구독 구조체 정의
// 채팅방에서 오고가는 메시지를 받음
type Subscription struct {
	Archive []Event		// 지금까지 쌓인 이벤트를 저장할 슬라이스
	New chan Event		// 새 이벤트가 생길 때마다 데이터를 받을 수 있도록
						// 이벤트 채널 생성
}

// 사용자 구조체 생성 함수
func NewUser(userId, userName string, userChan chan Event) User {
	return User{userId, userName, userChan}
}

// 이벤트 생성 함수
// Event 구조체 인스턴스 생성. 현재 시각으로 세팅
func NewEvent(evtType, userId, msg string) Event {
	return Event{evtType, userId, int(time.Now().Unix()), msg}
}

// 새로운 사용자가 들어왔을 때 이벤트를 구독할 함수
func Subscribe() Subscription {
	c := make(chan Subscription)	// 채널 생성
	subscribe <- c					// 구독 채널에 보냄
	return <-c
}

// 사용자가 나갔을 때 구독을 취소할 함수
func (s Subscription) Cancel() {
	unsubscribe <- s.New	// 구독 해지 채널에 보냄

	for {	// 무한 루프
		select {
		case _, ok := <-s.New:	// 채널에서 값을 꺼냄
			if !ok {			// 값을 모두 꺼냈으면 함수 빠져나옴
				return
			}
		// 채널에 값이 없다면 그냥 종료
		default:
			return
		}
	}
}

// 사용자가 회원 가입할 때 이벤트 발행
func RegisterUser(userId, userName string, userChan chan Event) {
	// register 채널에 현재 가입하는 사용자 정보 보냄
	register <- NewUser(userId, userName, userChan)

	fmt.Println(userId + " " + userName + " user registered")
}

// 사용자가 로그인할 때 이벤트 발행
func LoginUser(userId, userName string, userChan chan Event) {
	// login 채널에 현재 로그인하는 사용자 정보 보냄
	login <- NewUser(userId, userName, userChan)

	fmt.Println(userId + " " + userName + " user login requested")
}

// 현재 유저가 로그인 되어있는지 확인
func IsUserLogined(userId string) bool{
	// 확인 값을 받을 string 채널 생성
	isLogined := make(chan string)

	// loginCheck에 로그인 체크 요청 보냄
	loginCheck <- LoginCheckRequest{userId, isLogined}

	// 요청 받고 리턴
	return <-isLogined != "Anonymous"
}

// 현재 로그인 되어있는 사용자의 이름 리턴
func GetUserName(userId string) string{
	// 확인 값을 받을 string 채널 생성
	userNameChan := make(chan string)

	// loginCheck에 로그인 체크 요청 보냄
	loginCheck <- LoginCheckRequest{userId, userNameChan}
	
	// 요청 받고 리턴
	return <-userNameChan
}

// 사용자가 들어왔을 때 이벤트 발행
func Join(userName string) {
	// 사용자 이름으로 join 이벤트를 만들고 publish 채널로 보냄
	publish <- NewEvent("join", userName, "")

	fmt.Println(userName + " user joined")
}

// 사용자가 채팅 메시지를 보냈을 때 이벤트 발행
func Say(userName, message string) {
	// 사용자 이름, 메시지로 message 이벤트를 만들고 publish 채널로 보냄
	publish <- NewEvent("message", userName, message)

	fmt.Println(userName + " user said " + message)
}

// 사용자가 나갔을 때 이벤트 발행
func Leave(userName string) {
	// 사용자 이름으로 leave 이벤트를 만들고 publish 채널로 보냄
	publish <- NewEvent("leave", userName, "")

	fmt.Println(userName + " user left")
}

// 사용자 회원가입 및 로그인 처리할 함수
func UserManager() {
	var users []User
	loginedUsers := map[string]string{}

	for {	// 무한 루프
		select {
		// 회원 가입하면 register로 값 전송됨
		case newUser := <-register:
			// 회원가입된 유저 슬라이스에 추가
			users = append(users, newUser)

			fmt.Println(newUser.UserId + " " + newUser.UserName + " user appended")

		// 사용자가 로그인하면 login으로 값 전송됨
		case newUser := <-login:
			// 회원가입된 유저 맵에 추가
			loginedUsers[newUser.UserId] = newUser.UserName

			fmt.Println(newUser.UserId + " " + newUser.UserName + " user logined")

			// 유저에게 로그인 성공 메시지를 보냄
			newUser.UserChan <- NewEvent("loginsuccess", newUser.UserId, "")
			
			Join(newUser.UserName)	// 사용자가 채팅방에 들어왔다는 이벤트 발행
									// so.Id()는 socket.io의 세션 ID		

		// 유저가 로그인 되어있는지 확인 요청하면 loginCheck으로 값 전송됨
		case request := <-loginCheck:
			if val, ok := loginedUsers[request.UserId]; ok {
				request.Check <- val
			} else {
				request.Check <- "Anonymous"
			}
		}
	}
}

// 구독, 구독 해지, 발행된 이벤트를 처리할 함수
func Chatroom() {
	archive := list.New()		// 쌓인 이벤트를 저장할 연결 리스트
	subscribers := list.New()	// 구독자 목록을 저장할 연결 리스트

	for {	// 무한 루프
		select {
		case c := <-subscribe:	// 새로운 사용자가 들어왔을 때 -> subscribe에 값이 들어옴
			var events []Event	// 이벤트 슬라이스

			for e := archive.Front(); e != nil; e = e.Next() {	// 쌓인 이벤트가 있다면
				// events 슬라이스에 이벤트를 저장
				events = append(events, e.Value.(Event))
			}

			subscriber := make(chan Event, 10)	// 이벤트 채널 생성
			subscribers.PushBack(subscriber)	// 이벤트 채널을 구독자 목록에 추가

			c <- Subscription{events, subscriber}	// 구독 구조체 인스턴스 생성하고
													// 채널 c로 보냄

		case event := <-publish:	// 새 이벤트가 발행되었을 때 -> publish에 값이 들어옴 
									// Join, Say, Leave
			// 모든 사용자에게 이벤트 전달
			for e := subscribers.Front(); e != nil; e = e.Next() {
				// 구독자 목록에서 이벤트 채널을 꺼냄
				subscriber := e.Value.(chan Event)

				// 방금 받은 이벤트를 이벤트 채널에 보냄
				// 모든 사용자에게 이벤트를 전달해줌
				subscriber <- event
			}

			// 저장된 이벤트 개수가 20개가 넘으면
			if archive.Len() >= 20 {
				archive.Remove(archive.Front())	// 제일 앞에 있던 이벤트 삭제
			}
			archive.PushBack(event)	// 현재 이벤트를 저장

		case c := <-unsubscribe:	// 사용자가 나갔을 때 -> unsubscribe에 값이 들어옴 
			for e := subscribers.Front(); e != nil; e = e.Next() {
				// 구독자 목록에서 이벤트 채널을 꺼냄
				subscriber := e.Value.(chan Event)

				// 구독자 목록에 들어있는 이벤트와 채널 c가 같으면
				if subscriber == c {
					subscribers.Remove(e)	// 구독자 목록에서 삭제
					break
				}
			}
		}
	}
}

// 메인 함수
func main() {
	server, err := socketio.NewServer(nil)	// socket.io 초기화
											// nil : 모든 통신 방식 사용
	// 에러 처리
	if err != nil {
		log.Fatal(err)
	}

	// 사용자 관리 함수를 고루틴으로 실행
	go UserManager()

	go Chatroom() // 채팅방 처리 함수를 고루틴으로 실행

	// 웹 브라우저에서 socket.io로 접속했을 때 실행할 콜백 설정
	// On 함수로 각 상황마다 콜백 함수 실행
	server.On("connection", func(so socketio.Socket) {
		// 웹 브라우저가 접속되면
		s := Subscribe()	// 구독 처리

		fmt.Println(so.Id() + " user connected")

		for _, event := range s.Archive {	// 지금까지 쌓인 이벤트를
			so.Emit("event", event)			// 웹 브라우저로 접속한 사용자에게 보냄
											// event 메시지로 보냄
		}

		// string 채널 생성
		newMessages := make(chan string)

		// 웹 브라우저에서 보내오는 회원 가입 요청을 받을 수 있도록 콜백
		so.On("register", func(userName string) {
			// 받은 메시지를 newMessages 채널로 보냄
			RegisterUser(so.Id(), userName, s.New)
		})

		// 웹 브라우저에서 보내오는 로그인 요청 받을 수 있도록 콜백
		so.On("loginrequest", func(userName string) {
			// 받은 메시지를 newMessages 채널로 보냄
			LoginUser(so.Id(), userName, s.New)
		})

		// 웹 브라우저에서 보내오는 채팅 메시지를 받을 수 있도록 콜백
		so.On("message", func(msg string) {
			// 받은 메시지를 newMessages 채널로 보냄
			newMessages <- msg
		})

		// 웹 브라우저의 접속이 끊어졌을 때 콜백 설정
		so.On("disconnection", func() {
			// 로그인한 유저는 사용자가 채팅방에서 나갔다는 이벤트 발행
			if IsUserLogined(so.Id()) {
				Leave(GetUserName(so.Id()))
			}
			// 구독 취소
			s.Cancel()

			fmt.Println(so.Id() + " user disconnected")
		})

		// 고루틴 실행
		// 각 채널에 값이 들어왔을 때의 처리 함
		go func() {
			for {	 // 무한 루프
				select {
				case event := <-s.New:		// 채널에 이벤트가 들어오면
					so.Emit("event", event)	// 이벤트 데이터를 웹 브라우저에 보냄

				case msg := <-newMessages:	// 웹 브라우저에서 채팅 메시지를 보내오면
					Say(GetUserName(so.Id()), msg)		// 채팅 메시지 이벤트 발행
											// 사용자 아이디와 메시지 내용을 이벤트로 발행
				}
			}
		}()
	})

	http.Handle("/socket.io/", server)	// /socket.io/ 경로는 socket.io 인스턴스가 처리하도록 설정

	http.Handle("/", http.FileServer(http.Dir(".")))	// 현재 디렉토리를 파일 서버로 설정
														// "/"을 사용하여 index.html을 보여줌

	http.ListenAndServe(":8088", nil)			// 8088번 포트에서 웹 서버 실행
}
