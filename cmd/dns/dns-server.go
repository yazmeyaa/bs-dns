package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

// func startDNSServer(rc *redis.Client) {
// 	udpAddr, err := net.ResolveUDPAddr("udp", Address)
// 	if err != nil {
// 		log.Fatal("Failed to resolve UDP address", err)
// 	}

// 	udpConn, err := net.ListenUDP("udp", udpAddr)
// 	if err != nil {
// 		log.Fatal("Failed to bind to address", err)
// 	}
// 	defer udpConn.Close()

// 	buf := make([]byte, 512)
// 	for {
// 		_, source, err := udpConn.ReadFromUDP(buf)
// 		start := time.Now()
// 		log.Print("Ping\n")
// 		if err != nil {
// 			log.Println("Failed to receive data", err)
// 			continue
// 		}

// 		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))

// 		dnsHandler := dns.NewDNSHandler(rc)
// 		udpWriter := dns.NewUDPResponseWriter(udpConn, source)
// 		dnsHandler.HandleDNSQuery(ctx, buf, udpWriter)
// 		cancel()

// 		log.Printf("Processing time: %d ms", time.Since(start).Milliseconds())
// 	}
// }

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
		IPAddr:      "217.30.10.72",
	}

	r.Save(context.Background(), rc, db)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// go func() {
	// 	log.Printf("Started DNS server on %s", Address)
	// 	startDNSServer(rc)
	// }()

	go func() {
		log.Println("Starting DoH server...")
		handler := dns.NewDNSHandler(rc)
		startHTTPServer(handler)
	}()

	<-sigs
	log.Println("Shutting down servers...")
}
