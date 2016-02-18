package main

//http://blog.csdn.net/laotse/article/details/6296573
import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const (
	IP4    = 1
	Domain = 3
)

func main() {
	listen, err := net.Listen("tcp", ":4321")
	if err != nil {
		log.Println(err.Error())
		return
	}
	log.Println("socks5 is running... Port:4321")
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println(err.Error())
			continue
		}
		go socks5(conn)
	}

}

type address_enum byte

type rq struct {
	Version      byte         //固定为05
	Command      byte         //01说明是TCP
	Reserved     byte         //固定为00
	Address_type address_enum //域名还是IP，03是域名,01是IP
}

func socks5(conn net.Conn) {
	var err error
	var ver byte
	binary.Read(conn, binary.LittleEndian, &ver)
	//log.Println("ver:", ver)
	var method_count byte
	binary.Read(conn, binary.LittleEndian, &method_count)
	//log.Println("method_count:", method_count)
	methods := make([]byte, method_count)
	_, err = io.ReadFull(conn, methods)
	if err != nil {
		conn.Close()
		return
	}
	//log.Println("methods:", methods)

	_, err = conn.Write([]byte{5, 0}) //返回05 00  表示允许匿名代理
	if err != nil {
		log.Println(err.Error())
	}
	request := &rq{}
	binary.Read(conn, binary.LittleEndian, request)
	//log.Println("request:", request)
	if request.Version != 5 {
		conn.Close()
		log.Println("not socks5")
		return
	}
	if request.Command != 1 {
		conn.Close()
		log.Println("not tcp")
		return
	}
	var address string
	var port int16 //2个byte
	switch request.Address_type {
	case IP4:
		ip := make([]byte, 4) //固定占用4个byte
		binary.Read(conn, binary.LittleEndian, &ip)
		//log.Println("ip:", string(ip))
		//ip后面跟着端口port
		/*
			binary.Read(conn, binary.BigEndian, &port)
			log.Println("port:", port, "byteis:", byte(port))

		*/
		address = net.IP(ip).String()
	case Domain:
		var size byte //域名长度
		binary.Read(conn, binary.LittleEndian, &size)
		domain := make([]byte, size)
		io.ReadFull(conn, domain)
		//log.Println("domain:", string(domain))
		address = string(domain)
	default:
		log.Println("Invalid address type", request.Address_type)
		conn.Close()
		return
	}
	binary.Read(conn, binary.BigEndian, &port)
	//log.Println("port:", port)
	host := fmt.Sprintf("%s:%d", address, port)
	out, err := net.DialTimeout("tcp", host, 5*time.Second)
	log.Println("Connecting to ", host)
	if err != nil {
		conn.Close()
		log.Println("Connecting failed!", err.Error())
		return
	}
	_, err = conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
	//告知用户与目的主机连接成功
	if err != nil {
		log.Println(err.Error())
		conn.Close()
		return
	}

	now := time.Now().Add(time.Second * 60 * 2)
	conn.SetDeadline(now)
	out.SetDeadline(now)
	go inbound(conn, out)
	go outbound(conn, out)

}
func inbound(in net.Conn, out net.Conn) {
	_, err := io.Copy(in, out)
	if err != nil {
		in.Close()
		out.Close()
	}
}

func outbound(in net.Conn, out net.Conn) {
	_, err := io.Copy(out, in)
	if err != nil {
		in.Close()
		out.Close()
	}
}
