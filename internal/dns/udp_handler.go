package dns

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/bs-dns/internal/dns/answer"
	"github.com/yazmeyaa/bs-dns/internal/dns/header"
	"github.com/yazmeyaa/bs-dns/internal/dns/question"
	"github.com/yazmeyaa/bs-dns/internal/dns/records"
)

type ResponseWriter interface {
	WriteToResponse(data []byte) error
}

type DNSHandler struct {
	rc *redis.Client
}

type UDPResponseWriter struct {
	udpConn *net.UDPConn
	source  *net.UDPAddr
}

func NewUDPResponseWriter(udpConn *net.UDPConn, source *net.UDPAddr) *UDPResponseWriter {
	return &UDPResponseWriter{udpConn: udpConn, source: source}
}

func (w *UDPResponseWriter) WriteToResponse(data []byte) error {
	_, err := w.udpConn.WriteToUDP(data, w.source)
	return err
}

func NewDNSHandler(rc *redis.Client) *DNSHandler {
	return &DNSHandler{rc: rc}
}

func (h *DNSHandler) HandleDNSQuery(ctx context.Context, buf []byte, writer ResponseWriter) {
	if len(buf) < 12 {
		log.Println("Invalid DNS query, too small")
		return
	}

	/* Header has fixed length (12 bytes)  */
	hdr := header.ReadHeader(buf[:12])
	questions := make([]question.Question, hdr.QDCount)
	currentQuestionPos := 12

	log.Printf("Recieved data: %+v", buf)

	for x := 0; x < int(hdr.QDCount); x++ {
		/* 1. Read whole question, get offset in bytes. */
		q, offset := question.ReadQuestion(buf[currentQuestionPos:])
		/* 2. Add question to slice */
		questions[x] = q
		/* 3. Add offset to position pointer */
		currentQuestionPos += offset
	}

	var answers []answer.Answer

	for _, q := range questions {
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*4))
		defer cancel()
		r, err := records.GetDNSRecord(ctx, h.rc, q.QName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot resolve answer for domain name: [%s]. Error: %s", q.QName, err.Error())
			continue
		}
		answer := answer.Answer{
			Name:   q.QName,
			Data:   r.GetIPAddrBytes(),
			QType:  q.QType,
			QClass: q.QClass,
			TTL:    86400,
		}
		answers = append(answers, answer)
	}
	hdr.ANCount = uint16(len(answers))

	var res bytes.Buffer

	res.Write(hdr.Encode())
	for _, q := range questions {
		res.Write(q.Encode())
	}
	for _, a := range answers {
		res.Write(a.Encode())
	}

	writer.WriteToResponse(res.Bytes())
}

func (h *DNSHandler) HandleUDPQuery(udpConn *net.UDPConn, buf []byte) {
	_, source, err := udpConn.ReadFromUDP(buf)
	start := time.Now()
	log.Print("Ping\n")
	if err != nil {
		log.Println("Failed to receive data", err)
		return
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))

	dnsHandler := NewDNSHandler(h.rc)
	udpWriter := &UDPResponseWriter{udpConn: udpConn, source: source}
	dnsHandler.HandleDNSQuery(ctx, buf, udpWriter)

	cancel()

	log.Printf("Processing time: %d ms", time.Since(start).Milliseconds())
}
