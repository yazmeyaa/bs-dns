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
	clientConnections = make(map[string]*net.UDPConn)
	mu                sync.Mutex
)

func handleClient(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, n int) {
	remoteAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(remoteHost, remotePort))
	if err != nil {
		log.Printf("Error resolving remote address: %v", err)
		return
	}

	mu.Lock()
	remoteConn, exists := clientConnections[clientAddr.String()]
	if !exists {
		remoteConn, err = net.DialUDP("udp", nil, remoteAddr)
		if err != nil {
			log.Printf("Error connecting to remote server: %v", err)
			mu.Unlock()
			return
		}
		clientConnections[clientAddr.String()] = remoteConn

		go func(addr string, conn *net.UDPConn) {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			delete(clientConnections, addr)
			mu.Unlock()
			conn.Close()
		}(clientAddr.String(), remoteConn)
	}
	mu.Unlock()

	_, err = remoteConn.Write(data[:n])
	if err != nil {
		log.Printf("Error sending data to remote server: %v", err)
		return
	}

	buf := make([]byte, 2048)
	n, err = remoteConn.Read(buf)
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
		log.Printf("Received %d bytes from client %s", n, clientAddr.String())

		go handleClient(conn, clientAddr, buf, n)
	}
}

func main() {
	startBSProxyServer()
}
