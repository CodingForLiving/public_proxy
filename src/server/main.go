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
    "io/ioutil"
    "time"
)

type Proxy struct {
    ID       int
    Key      int
    Conn     *net.Conn `json:"-"`
	DestAddr string
	FromBind string
	FromAddr string
	ToBind   string
	ToAddr   string
    Chan     chan string `json:"-"`
}

type Config struct {
    BindAddr string
    ProxyArray []Proxy
}

var cfg *Config = &Config{}

var lock *sync.Mutex
var proxyMap map[int]*Proxy = map[int]*Proxy{}

func main_sock(addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
        fmt.Println("bind error:",err.Error())
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
    length := len(data)
    conn.Write([]byte{byte(length>>8),byte(length & 0xff)})
	conn.Write(data)
}

func handle_main_sock_conn(conn net.Conn) {
	fmt.Println("proxy client conn from ",conn.RemoteAddr())
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(time.Duration(5) * time.Second))
	length_array := make([]byte, 2)
	n, err := conn.Read(length_array)
	if err != nil {
        fmt.Println("handle_main_sock_conn read len error:",err.Error())
		return
	}

	if n != 2 {
        fmt.Println("handle_main_sock_conn read len error")
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

    fmt.Println("proxy client id: ",id)
	for {
		_, ok := <-p.Chan
		fmt.Println("发送连接请求给客户端")
		if ok {
			send(conn, map[string]string{
			    "cmd": "connect",
			    "dest_addr": p.DestAddr,
			    "to_addr": p.ToAddr,
			})
		}
	}
}

func bridge_listen(client_addr string, proxy_addr string,ch *chan string) {
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
		if err != nil {
			fmt.Println(err)
			break
		}
		// 通知客户端代理
        fmt.Println("通知客户端代理")
        *ch <- "ok"
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

    f,err := os.Open("server.cfg")
    if err != nil {
        fmt.Println("read cfg error:",err.Error())
        return
    }
    defer f.Close()

    if bytes,err := ioutil.ReadAll(f);err == nil {
        err = json.Unmarshal(bytes,cfg)
        if err != nil {
            fmt.Println("parse cfg error:",err.Error())
            return
        }
    }

    lock = new(sync.Mutex)
    for _,v := range cfg.ProxyArray {
        p := &v
        p.Chan = make(chan string, 5)
	    proxyMap[p.ID] = p
    }

	go main_sock(cfg.BindAddr)

	for _, v := range proxyMap {
		go bridge_listen(v.FromBind, v.ToBind, &v.Chan)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	fmt.Println("server shutdown")
}
