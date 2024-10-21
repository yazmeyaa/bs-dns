package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/bs-dns/internal/config"
	"github.com/yazmeyaa/bs-dns/internal/dns"
	"github.com/yazmeyaa/bs-dns/internal/dns/records"
	"github.com/yazmeyaa/bs-dns/pkg/db"
	_redis "github.com/yazmeyaa/bs-dns/pkg/redis"
)

const Address = "0.0.0.0:53"

func migrateDB(db *sql.DB) error {
	goose.SetDialect("sqlite3")
	migrationsDir := "pkg/db/migrations"
	if err := goose.Up(db, migrationsDir); err != nil {
		return err
	}
	log.Println("Successfully migrated DB")
	return nil
}

func initServices() (*sql.DB, *redis.Client, error) {
	db, err := db.InitDB()
	if err != nil {
		return nil, nil, err
	}

	if err := migrateDB(db); err != nil {
		return nil, nil, err
	}

	config, err := config.New()
	if err != nil {
		return nil, nil, err
	}

	rc, err := _redis.InitRedisConnection(context.Background(), config)
	if err != nil {
		return nil, nil, err
	}

	return db, rc, nil
}

func startDNSServer(rc *redis.Client) {
	udpAddr, err := net.ResolveUDPAddr("udp", Address)
	if err != nil {
		log.Fatal("Failed to resolve UDP address", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal("Failed to bind to address", err)
	}
	defer udpConn.Close()

	buf := make([]byte, 512)
	for {
		_, source, err := udpConn.ReadFromUDP(buf)
		start := time.Now()
		log.Print("Ping\n")
		if err != nil {
			log.Println("Failed to receive data", err)
			continue
		}

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))

		dnsHandler := dns.NewDNSHandler(rc)
		udpWriter := dns.NewUDPResponseWriter(udpConn, source)
		dnsHandler.HandleDNSQuery(ctx, buf, udpWriter)
		cancel()

		log.Printf("Processing time: %d ms", time.Since(start).Milliseconds())
	}
}

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

func startHTTPServer(dnsHandler *dns.DNSHandler) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /dns-query", dnsHandler.HttpHandler)
	log.Println("Starting HTTP server on :8080")
	err := http.ListenAndServeTLS(":8080", "cert/server.crt", "cert/server.key", mux)
	if err != nil {
		log.Fatal("Failed to start HTTP server:", err)
	}
}

func main() {
	db, rc, err := initServices()
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	r := records.DNSRecord{
		Label:       "Brawl stars",
		Description: "Brawl stars game traffic",
		Name:        "game.brawlstarsgame.com",
		IPAddr:      "192.168.1.79",
	}

	r.Save(context.Background(), rc, db)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Started DNS server on %s", Address)
		startDNSServer(rc)
	}()

	go func() {
		log.Println("Starting DoH server...")
		handler := dns.NewDNSHandler(rc)
		startHTTPServer(handler)
	}()

	go func() {
		log.Println("Starting Brawl Stars proxy server...")
		startBSProxyServer()
	}()

	<-sigs
	log.Println("Shutting down servers...")
}
