package main

import (
	"fmt"
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string // 消息通道
	conn   net.Conn
	server *Server
}

func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}

	// 创建一个goroutine，启动监听当前user的channel
	go user.ListenMessage()

	return user
}

// 用户上线业务
func (this *User) Online() {

	// 用户上线，将用户加入onlinemap中
	this.server.mapLock.Lock() // 协程会调用共有的map表，防止操作失误，需要上锁

	this.server.OnlineMap[this.Name] = this

	this.server.mapLock.Unlock()

	this.server.BroadCast(this, "已上线")
}

// 用户下线业务
func (this *User) Offline() {
	// 用户下线，将用户从onlinemap中删除
	this.server.mapLock.Lock() // 协程会调用共有的map表，防止操作失误，需要上锁

	delete(this.server.OnlineMap, this.Name)

	this.server.mapLock.Unlock()

	this.server.BroadCast(this, "已下线")
}

// 给当前用户发消息
func (this *User) SendMsg(msg string) {
	this.conn.Write([]byte(msg))
}

// 用户处理消息业务
func (this *User) DoMessage(msg string) {
	// this.server.BroadCast(this, msg)
	if msg == "who" {
		// 查询当前在线用户有哪些

		this.server.mapLock.Lock()

		for _, user := range this.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线……\n"
			this.SendMsg(onlineMsg)
		}

		this.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[0:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]

		// 判断name是否存在
		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMsg("该用户名已经被占用……\n")
		} else {
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.Name = newName
			this.server.mapLock.Unlock()

			this.SendMsg("用户名更新完成……\n")
		}

	} else if len(msg) > 4 && msg[0:3] == "to|" {
		// 格式 to|张三|消息……
		toName := strings.Split(msg, "|")[1]
		toMsg := this.Name + ":" + strings.Split(msg, "|")[2]

		if toName == "" {
			this.SendMsg("消息格式不正确，停止发送\n")
			return
		}

		toUser, ok := this.server.OnlineMap[toName]
		if !ok {
			this.SendMsg("用户名不存在，停止发送\n")
			return
		}

		toUser.SendMsg(toMsg)
		this.SendMsg("消息发送成功")

	} else {
		this.server.BroadCast(this, msg)
	}

}

// 监听当前user 的channel ，有消息则直接发给对端client
//
//func (u *User) ListenMessage() {
//	//当u.C通道关闭后，不再进行监听并写入信息
//	for msg := range u.C {
//		_, err := u.conn.Write([]byte(msg + "\n"))
//		if err != nil {
//			panic(err)
//		}
//	}
//	//不监听后关闭conn，conn在这里关闭最合适
//	err := u.conn.Close()
//	if err != nil {
//		panic(err)
//	}
//}

func (this *User) ListenMessage() {
	for {
		msg := <-this.C

		_, err := this.conn.Write([]byte(msg + "\n"))
		if err != nil {
			fmt.Println(err)
		}
	}
}
