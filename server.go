package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	//在线用户信息表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex // 对用户信息表上一个锁， RWMutex 表示这是一个读写锁

	// 消息广播channel
	Message chan string
}

// 创建一个Server的接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}

	return server
}

// 以下全是Server类的方法

// 监听Message广播消息的channel，一旦发现有消息了就要广播给全部用户
func (this *Server) ListenMessage() {
	for {
		msg := <-this.Message

		// 将msg广播给所有在线用户
		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()

	}
}

// 发送广播消息的方法
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg

	this.Message <- sendMsg // 通道传输 都是需要使用 <- 或者 ->
}

func (this *Server) Handler(conn net.Conn) {
	// 业务逻辑写在这里
	//fmt.Printf("链接建立成功！！")

	// 用户上线，启动用户上线方法
	user := NewUser(conn, this)
	user.Online()

	// 监听用户是否活跃的channel
	isLive := make(chan bool)

	// 接收用户发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 { // 用户断开连接
				user.Offline()
				return
			}

			if err != nil && err != io.EOF {
				fmt.Printf("Conn Read err: ", err)
				user.Offline()
				return
			}

			msg := string(buf[:n-1]) //提取用户的信息（去除 ‘\n’）

			// 将得到的消息进行流转处理
			user.DoMessage(msg)

			// 激活心跳，代表活跃用户
			isLive <- true
		}
	}()

	// 使得当前的handle阻塞
	select {
	case <-isLive:
		// 当前用户是活跃的，重置定时器
		// 不做任何处理,为了激活select，更新下面的定时器。
	case <-time.After(time.Second * 120):
		// 用户长时间未操作，超时
		// 强制关闭该用户

		user.SendMsg("长时间没操作，你被踢了！")
		user.Offline()

		// 销毁资源
		close(user.C) // 销毁管道资源
		//conn.Close()  // 删除通讯连接

		// 退出当前goroutine （handle）
		//close(isLive)
		return //runtime.Goexit()
	}
}

// 启动服务器的接口
func (this *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))

	if err != nil {
		fmt.Printf("net.Listen err: ", err)
	}
	// close listen socket
	defer listener.Close()

	// 启动监听Message的goroutine
	go this.ListenMessage()

	// 循环阻塞 监听端口，等待user连接
	for {
		// accept
		conn, err := listener.Accept()

		if err != nil {
			fmt.Printf("listener accept err: ", err)
			continue
		}

		// do handler
		go this.Handler(conn)

	}

}
