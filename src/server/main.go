package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var account string = ""
var listenaddr string = "3721"

type Proxy struct {
	ID       int
	Key      int
	Conn     *net.Conn
	DestAddr string
	FromAddr string
	ToAddr   string
	Chan     chan string
}

var lock *sync.Mutex
var proxyMap map[int]*Proxy = map[int]*Proxy{}

func main_sock(addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			continue
		}
		go handle_main_sock_conn(c)
	}
}

func send(conn net.Conn, i interface{}) {
	data, err := json.Marshal(i)
	if err != nil {
		return
	}
	conn.Write(data)
}

func handle_main_sock_conn(conn net.Conn) {
	defer conn.Close()
	length_array := make([]byte, 2)
	n, err := conn.Read(length_array)
	if n != 2 {
		return
	}

	length := int(length_array[0])<<8 + int(length_array[1])
	data := make([]byte, length)

	n, err = conn.Read(data)
	if n != length {
		return
	}

	req := map[string]int{}
	err = json.Unmarshal(data, &req)
	if err != nil {
		return
	}

	id, ok1 := req["id"]
	key, ok2 := req["key"]
	if !ok1 || !ok2 {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	p, ok := proxyMap[id]
	if !ok || key != p.Key {
		return
	}
	p.Conn = &conn

	for {
		_, ok := <-p.Chan
		if ok {
			send(conn, map[string]string{"cmd": "connect", "dest_addr": p.DestAddr})
		}
	}
}

func bridge_listen(client_addr string, proxy_addr string) {
	cl, err := net.Listen("tcp", client_addr)
	if err != nil {
		return
	}
	pl, err1 := net.Listen("tcp", proxy_addr)
	if err1 != nil {
		return
	}
	for {
		client_c, err := cl.Accept()
		if err == nil {
			break
		}
		// 通知客户端代理
		proxy_c, err := pl.Accept()
		go bridge(client_c, proxy_c)
		go bridge(proxy_c, client_c)
	}
}

func bridge(from net.Conn, to net.Conn) {
	io.Copy(from, to)
}

func main() {
	fmt.Println("server start")
	lock = new(sync.Mutex)
	proxyMap[1] = &Proxy{
		ID:       1,
		Key:      1,
		DestAddr: ":5000",
		FromAddr: ":5000",
		ToAddr:   ":5001",
		Chan:     make(chan string, 5),
	}

	go main_sock(listenaddr)

	for _, v := range proxyMap {
		go bridge_listen(v.FromAddr, v.ToAddr)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	fmt.Println("server shutdown")
}
