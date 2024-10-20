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
	"github.com/yazmeyaa/bs-dns/internal/config"
	"github.com/yazmeyaa/bs-dns/internal/dns/answer"
	"github.com/yazmeyaa/bs-dns/internal/dns/header"
	"github.com/yazmeyaa/bs-dns/internal/dns/question"
	"github.com/yazmeyaa/bs-dns/internal/dns/records"
	"github.com/yazmeyaa/bs-dns/internal/redis"
	"github.com/yazmeyaa/bs-dns/pkg/db"
)

const Address = "0.0.0.0:53"

var nameToIP = make(map[string][]byte)

func migrateDB(db *sql.DB) error {
	goose.SetDialect("sqlite3")
	const migrationsDir string = "pkg/db/migrations"

	if err := goose.Up(db, migrationsDir); err != nil {
		return err
	}

	log.Println("Successfull migrated DB")

	return nil
}

func main() {
	db, err := db.InitDB()
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	if err := migrateDB(db); err != nil {
		log.Fatal(err.Error())
		return
	}
	config, err := config.New()

	rc, err := redis.InitRedisConnection(context.Background(), config)

	if err != nil {
		log.Fatal(err.Error())
		return
	}

	r := records.DNSRecord{
		Name:        "1.0.0.127.in-addr.arpa",
		Label:       "localhost",
		Description: "Localhost address",
		IPAddr:      "127.0.0.1",
	}

	err = r.Save(context.Background(), rc, db)
	if err != nil {
		log.Println(err.Error())
	}
	udpAddr, err := net.ResolveUDPAddr("udp", Address)
	if err != nil {
		log.Fatal("failed to resolve udp address", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal("failed to to bind to address", err)
	}
	defer udpConn.Close()

	log.Printf("started server on %s", Address)
	buf := make([]byte, 512)
	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		start := time.Now()
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
		log.Printf("QUESTION QTYPE: %d", q.QType)

		var res bytes.Buffer

		if q.QType == 28 {
			h.ResponseCode = header.RCODE_NOT_IMPLEMENTED
			res.Write(h.Encode())
			res.Write(q.Encode())

			_, err = udpConn.WriteToUDP(res.Bytes(), source)
			if err != nil {
				log.Println("Failed to send response:", err)
			}
			continue
		}

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
		defer cancel()
		record, err := records.GetDNSRecord(ctx, rc, q.QName)

		if err != nil {
			if errors.Is(records.ErrRecordNotFound, err) {
				log.Printf("Record with domain name [%s] not found", q.QName)
			} else {
				log.Printf("Error while getting record: %s", err.Error())
			}
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
			Data:   record.GetIPAddrBytes(),
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

		log.Printf("Resolved name: %s => %s", q.QName, record.IPAddr)
		log.Printf("PROCESSING TIME: %d ms", time.Since(start).Milliseconds())
	}

}
