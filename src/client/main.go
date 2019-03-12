package main

import(
    "fmt"
    "net"
    "encoding/json"
    "io"
    "os"
    "os/signal"
    "syscall"
    "flag"
)

func send(conn net.Conn, i interface{}) {
	data, err := json.Marshal(i)
	if err != nil {
		return
	}
    length := len(data)
    conn.Write([]byte{byte(length>>8),byte(length & 0xff)})
	conn.Write(data)
}

func proxy(addr string,id,key int) error {
    conn,err := net.Dial("tcp",addr)
    if err != nil {
        return err
    }

    hello := map[string]interface{}{
        "id": id,
        "key": key,
    }

    send(conn,hello)

    for {
        len_array := make([]byte,2)
        n,err := conn.Read(len_array)
        fmt.Println(n,err)
        if err != nil || n != 2{
            break
        }
        length := (int(len_array[0])<<8) + int(len_array[1])
        data := make([]byte,length)
        n,err = conn.Read(data)
        if err != nil {
            break
        }
        msg := map[string]interface{}{}
        err = json.Unmarshal(data,&msg)
        if err != nil {
            break
        }
        fmt.Println(msg)
        destAddr,ok1 := msg["dest_addr"]
        toAddr,ok2 := msg["to_addr"]
        if !ok1 || !ok2 {
            break
        }
        err = bridge(destAddr.(string),toAddr.(string))
        if err != nil {
            fmt.Println("build bridge error:",err)
        }
    }
    return nil
}

func bridge(destAddr,toAddr string)error{
    dest_conn, err1 := net.Dial("tcp",destAddr)
    if err1 != nil {
        return err1
    }
    to_conn, err2 := net.Dial("tcp",toAddr)
    if err1 != nil {
        return err2
    }
    go func (){io.Copy(dest_conn,to_conn)}()
    go func (){io.Copy(to_conn,dest_conn)}()
    return nil
}

func main(){
    fmt.Println("client start")
    id := flag.Int("id",1,"id")
    key := flag.Int("key",1,"key")
    host := flag.String("host","a.shortlife.top","host")
    port := flag.String("port","3721","port")
    go proxy(*host+":"+*port,*id,*key)

    ch := make(chan os.Signal)
    signal.Notify(ch,syscall.SIGINT,syscall.SIGTERM)

    <-ch

    fmt.Println("client shutdown")
}
