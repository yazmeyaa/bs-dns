package main

import (
	"log"
	"net"
	"sync"
	"time"
)

const (
	localPort  = ":9339"
	remoteHost = "game.brawlstarsgame.com"
	remotePort = "9339"
)

var (
	clientConnections = sync.Map{}
)

func handleClient(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, n int) {
	remoteAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(remoteHost, remotePort))
	if err != nil {
		log.Printf("Error resolving remote address: %v", err)
		return
	}

	key := clientAddr.String()

	remoteConn, _ := clientConnections.LoadOrStore(key, func() *net.UDPConn {
		conn, err := net.DialUDP("udp", nil, remoteAddr)
		if err != nil {
			log.Printf("Error connecting to remote server: %v", err)
			return nil
		}

		go func(key string, conn *net.UDPConn) {
			time.Sleep(5 * time.Minute)
			clientConnections.Delete(key)
			conn.Close()
		}(key, conn)

		return conn
	}())

	if remoteConn == nil {
		return
	}

	_, err = remoteConn.(*net.UDPConn).Write(data[:n])
	if err != nil {
		log.Printf("Error sending data to remote server: %v", err)
		return
	}

	remoteConn.(*net.UDPConn).SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 2048)
	n, err = remoteConn.(*net.UDPConn).Read(buf)
	if err != nil {
		log.Printf("Error receiving data from remote server: %v", err)
		return
	}

	_, err = conn.WriteToUDP(buf[:n], clientAddr)
	if err != nil {
		log.Printf("Error sending data to client: %v", err)
	}
}

func startBSProxyServer() {
	listenAddr, err := net.ResolveUDPAddr("udp", localPort)
	if err != nil {
		log.Fatalf("Error resolving local address: %v", err)
	}

	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		log.Fatalf("Error starting UDP server: %v", err)
	}
	defer conn.Close()
	log.Printf("Listening on %s", localPort)

	buf := make([]byte, 2048)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error receiving data from client: %v", err)
			continue
		}
		dataCopy := make([]byte, n)
		copy(dataCopy, buf[:n])
		go handleClient(conn, clientAddr, dataCopy, n)
	}
}

func main() {
	startBSProxyServer()
}
