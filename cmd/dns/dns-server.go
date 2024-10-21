package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/bs-dns/internal/config"
	"github.com/yazmeyaa/bs-dns/internal/dns/answer"
	"github.com/yazmeyaa/bs-dns/internal/dns/header"
	"github.com/yazmeyaa/bs-dns/internal/dns/question"
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

func handleDNSQuery(ctx context.Context, buf []byte, source *net.UDPAddr, udpConn *net.UDPConn, rc *redis.Client) {
	if len(buf) < 12 {
		log.Println("Invalid DNS query, too small")
		return
	}

	h := header.ReadHeader(buf[:12])
	h.IsResponse = true
	q, _ := question.ReadQuestion(buf[12:])
	h.QDCount = 1

	var res bytes.Buffer
	if q.QType == 28 {
		h.ResponseCode = header.RCODE_NOT_IMPLEMENTED
		res.Write(h.Encode())
		res.Write(q.Encode())
		udpConn.WriteToUDP(res.Bytes(), source)
		return
	}

	record, err := records.GetDNSRecord(ctx, rc, q.QName)
	if err != nil {
		if errors.Is(err, records.ErrRecordNotFound) {
			log.Printf("Record with domain name [%s] not found", q.QName)
		} else {
			log.Printf("Error while getting record: %s", err.Error())
		}
		res.Write(h.Encode())
		res.Write(q.Encode())
		udpConn.WriteToUDP(res.Bytes(), source)
		return
	}

	ans := answer.Answer{
		Name:   q.QName,
		QType:  question.TYPE_HOST,
		QClass: question.CLASS_INTERNET,
		TTL:    0,
		Data:   record.GetIPAddrBytes(),
	}
	h.ANCount++
	h.ResponseCode = header.RCODE_NO_ERROR

	res.Write(h.Encode())
	res.Write(q.Encode())
	res.Write(ans.Encode())

	udpConn.WriteToUDP(res.Bytes(), source)
	log.Printf("Resolved name: %s => %s", q.QName, record.IPAddr)
}

func startDNSServer(udpConn *net.UDPConn, rc *redis.Client, db *sql.DB) {
	buf := make([]byte, 512)
	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		start := time.Now()
		if err != nil {
			log.Println("Failed to receive data", err)
			continue
		}

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		handleDNSQuery(ctx, buf[:size], source, udpConn, rc)
		cancel()

		log.Printf("Processing time: %d ms", time.Since(start).Milliseconds())
	}
}

func main() {
	db, rc, err := initServices()
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	udpAddr, err := net.ResolveUDPAddr("udp", Address)
	if err != nil {
		log.Fatal("Failed to resolve UDP address", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal("Failed to bind to address", err)
	}
	defer udpConn.Close()

	log.Printf("Started server on %s", Address)
	startDNSServer(udpConn, rc, db)
}
