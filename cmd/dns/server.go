package main

import (
	"bytes"
	"fmt"
	"log"
	"net"

	"github.com/yazmeyaa/bs-dns/internal/header"
	"github.com/yazmeyaa/bs-dns/internal/question"
)

const Address = "127.0.0.1:53"

func main() {
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
		log.Printf("Received %d bytes from %s", size, source.String())
		if err != nil {
			log.Println("failed to receive data", err)
			continue
		}

		log.Printf("received %d bytes from %s", size, source.String())

		if size < 12 {
			log.Println("invalid DNS query, too small")
			continue
		}

		header := header.ReadHeader(buf[:12])
		log.Printf("ID: %d; QR: %t; QDCount: %d\n", header.ID, header.IsResponse, header.QDCount)
		header.IsResponse = true

		question, _ := question.ReadQuestion(buf[12:])

		var res bytes.Buffer
		res.Write(header.Encode())
		res.Write(question.Encode())

		_, err = udpConn.WriteToUDP(res.Bytes(), source)
		if err != nil {
			log.Println("Failed to send response:", err)
		}
		log.Printf("Response sent\n")
	}

}
