package main

import (
	"log"
	"net"
)

func startBSProxyServer() {
	const (
		localPort  = ":9339"
		remoteHost = "game.brawlstarsgame.com"
		remotePort = "9339"
	)

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

	remoteAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(remoteHost, remotePort))
	if err != nil {
		log.Fatalf("Error resolving remote address: %v", err)
	}

	buf := make([]byte, 2048)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error receiving data from client: %v", err)
			continue
		}
		log.Printf("Received %d bytes from client %s", n, clientAddr.String())

		remoteConn, err := net.DialUDP("udp", nil, remoteAddr)
		if err != nil {
			log.Printf("Error connecting to remote server: %v", err)
			continue
		}
		_, err = remoteConn.Write(buf[:n])
		if err != nil {
			log.Printf("Error sending data to remote server: %v", err)
			remoteConn.Close()
			continue
		}

		n, err = remoteConn.Read(buf)
		if err != nil {
			log.Printf("Error receiving data from remote server: %v", err)
			remoteConn.Close()
			continue
		}
		log.Printf("Received %d bytes from remote server", n)

		_, err = conn.WriteToUDP(buf[:n], clientAddr)
		if err != nil {
			log.Printf("Error sending data to client: %v", err)
		}

		remoteConn.Close()
	}
}

func main() {
	startBSProxyServer()
}
