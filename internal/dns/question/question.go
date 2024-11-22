package question

import (
	"encoding/binary"
	"log"
)

type Class uint16

const (
	_ Class = iota
	CLASS_INTERNET
	CLASS_CSNET
	CLASS_CHAOS
	CLASS_HESIOD
)

type Type uint16

const (
	_ Type = iota
	TYPE_HOST
	TYPE_NAME_SERVER
	TYPE_MAIL_DESTINATION
	TYPE_MAIL_FORWARDER
	TYPE_CANONICAL_NAME
	TYPE_START_OF_A_ZONE_OF_AUTHORITY
	TYPE_MAILBOX_DOMAIN
	TYPE_MAIL_GROUP_MEMBER
	TYPE_MAIL_RENAME_DOMAIN
	TYPE_NULL_RR
	TYPE_WELL_KNOWN_SERVICE
	TYPE_DOMAIN_NAME_POINTER
	TYPE_HOST_INFORMATION
	TYPE_MAILBOX_LIST_INFORMATION
	TYPE_MAIL_EXCHANGE
	TYPE_TEXT_STRINGS
	TYPE_HOST_V6 = 28
)

type Question struct {
	QName  string
	QType  Type
	QClass Class
}

func ReadLabel(data []byte, offset *int) string {
	var labels []byte
	for {
		if *offset >= len(data) {
			log.Println("Error: Offset exceeds available data")
			break
		}
		length := int(data[*offset])
		*offset++
		if length == 0 {
			break
		}

		if *offset+length > len(data) {
			log.Println("Error: Label length exceeds available data")
			break
		}
		labels = append(labels, data[*offset:*offset+length]...)
		labels = append(labels, '.')
		*offset += length
	}
	return string(labels[:len(labels)-1])
}

func ReadQuestion(data []byte) (Question, int) {
	var q Question
	offset := 0

	q.QName = ReadLabel(data, &offset)

	q.QType = Type(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2

	q.QClass = Class(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2

	return q, offset
}

func (q *Question) Encode() []byte {
	var encoded []byte

	labels := SplitQName(q.QName)
	for _, label := range labels {
		encoded = append(encoded, byte(len(label)))
		encoded = append(encoded, label...)
	}
	encoded = append(encoded, 0x00)

	qTypeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(qTypeBytes, uint16(q.QType))
	encoded = append(encoded, qTypeBytes...)

	qClassBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(qClassBytes, uint16(q.QClass))
	encoded = append(encoded, qClassBytes...)

	return encoded
}

func SplitQName(qname string) []string {
	var labels []string
	start := 0

	for i := 0; i < len(qname); i++ {
		if qname[i] == '.' {
			labels = append(labels, qname[start:i])
			start = i + 1
		}
	}
	if start < len(qname) {
		labels = append(labels, qname[start:])
	}

	return labels
}
