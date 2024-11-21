package dns

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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

	hdr := header.ReadHeader(buf[:12])
	hdr.IsResponse = true
	q, _ := question.ReadQuestion(buf[12:])
	hdr.QDCount = 1

	var res bytes.Buffer
	if q.QType == 28 {
		hdr.ResponseCode = header.RCODE_NOT_IMPLEMENTED
		res.Write(hdr.Encode())
		res.Write(q.Encode())
		writer.WriteToResponse(res.Bytes())
		return
	}

	record, err := records.GetDNSRecord(ctx, h.rc, q.QName)
	if err != nil {

		var query bytes.Buffer

		res.Write(hdr.Encode())
		res.Write(q.Encode())
		fwd, err := h.forwardQuery(query.Bytes())

		if err != nil {
			if errors.Is(err, records.ErrRecordNotFound) {
				log.Printf("Record with domain name [%s] not found", q.QName)
			} else {
				log.Printf("Error while getting record: %s", err.Error())
			}
			res.Write(hdr.Encode())
			res.Write(q.Encode())
			writer.WriteToResponse(res.Bytes())
			return
		}

		writer.WriteToResponse(fwd)
		return
	}

	ans := answer.Answer{
		Name:   q.QName,
		QType:  question.TYPE_HOST,
		QClass: question.CLASS_INTERNET,
		TTL:    0,
		Data:   record.GetIPAddrBytes(),
	}
	hdr.ANCount++
	hdr.ResponseCode = header.RCODE_NO_ERROR

	res.Write(hdr.Encode())
	res.Write(q.Encode())
	res.Write(ans.Encode())

	log.Printf("Resolved name: %s => %s", q.QName, record.IPAddr)
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

func (h *DNSHandler) forwardQuery(query []byte) ([]byte, error) {
	upstreamServer := "1.1.1.1:53"

	conn, err := net.Dial("udp", upstreamServer)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to upstream DNS server: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write(query)
	if err != nil {
		return nil, fmt.Errorf("failed to send query to upstream server: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	response := make([]byte, 512)
	n, err := conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from upstream server: %w", err)
	}

	return response[:n], nil
}
