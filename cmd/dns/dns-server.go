package main

import (
	"bytes"
	"fmt"
	"log"
	"net"

	"github.com/yazmeyaa/bs-dns/internal/db"
	"github.com/yazmeyaa/bs-dns/internal/dns/answer"
	"github.com/yazmeyaa/bs-dns/internal/dns/header"
	"github.com/yazmeyaa/bs-dns/internal/dns/question"
)

const Address = "0.0.0.0:53"

var nameToIP = make(map[string][]byte)

func main() {
	nameToIP["game.brawlstars.com"] = []byte{12, 34, 56, 78}
	db, err := db.InitDB()
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	log.Printf("%+v", db)

	udpAddr, err := net.ResolveUDPAddr("udp", Address)
	if err != nil {
		log.Fatal("failed to resolve udp address", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal("failed to to bind to address", err)
	}
	fmt.Printf("IP: %s, PORT: %d\n", udpAddr.IP.String(), udpAddr.Port)
	defer udpConn.Close()

	log.Printf("started server on %s", Address)
	buf := make([]byte, 512)
	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		log.Printf("\n==>Ping\n")
		if err != nil {
			log.Println("failed to receive data", err)
			continue
		}

		if size < 12 {
			log.Println("invalid DNS query, too small")
			continue
		}

		h := header.ReadHeader(buf[:12])
		h.IsResponse = true
		h.ANCount = 0
		q, _ := question.ReadQuestion(buf[12:])
		h.QDCount = 1

		var res bytes.Buffer

		ip, ok := nameToIP[q.QName]
		if !ok {
			log.Println("No record found for:", q.QName)
			h.ResponseCode = header.RCODE_NAME_ERROR
			res.Write(h.Encode())
			res.Write(q.Encode())
			_, err = udpConn.WriteToUDP(res.Bytes(), source)
			if err != nil {
				log.Println("Failed to send response:", err)
			}
			continue
		}

		ans := answer.Answer{
			Name:   q.QName,
			QType:  question.TYPE_HOST,
			QClass: question.CLASS_INTERNET,
			TTL:    0,
			Data:   ip,
		}
		h.ANCount++
		h.ResponseCode = header.RCODE_NO_ERROR

		res.Write(h.Encode())
		res.Write(q.Encode())
		res.Write(ans.Encode())

		_, err = udpConn.WriteToUDP(res.Bytes(), source)
		if err != nil {
			log.Println("Failed to send response:", err)
		}

		log.Printf("Resolved name: %s => %d.%d.%d.%d", q.QName, ip[0], ip[1], ip[2], ip[3])
	}

}
