package answer

import (
	"encoding/binary"

	"github.com/yazmeyaa/bs-dns/internal/question"
)

type Answer struct {
	Name   string
	QType  question.Type
	QClass question.Class
	TTL    uint32
	Data   []byte
}

func ReadAnswer(data []byte, offset *int) Answer {
	var ans Answer

	ans.Name = question.ReadLabel(data, offset)

	ans.QType = question.Type(binary.BigEndian.Uint16(data[*offset : *offset+2]))
	*offset += 2

	ans.QClass = question.Class(binary.BigEndian.Uint16(data[*offset : *offset+2]))
	*offset += 2

	ans.TTL = binary.BigEndian.Uint32(data[*offset : *offset+4])
	*offset += 4

	dataLength := binary.BigEndian.Uint16(data[*offset : *offset+2])
	*offset += 2

	ans.Data = data[*offset : *offset+int(dataLength)]
	*offset += int(dataLength)

	return ans
}

func (a *Answer) Encode() []byte {
	var encoded []byte

	labels := question.SplitQName(a.Name)
	for _, label := range labels {
		encoded = append(encoded, byte(len(label)))
		encoded = append(encoded, label...)
	}
	encoded = append(encoded, 0x00)

	qTypeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(qTypeBytes, uint16(a.QType))
	encoded = append(encoded, qTypeBytes...)

	qClassBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(qClassBytes, uint16(a.QClass))
	encoded = append(encoded, qClassBytes...)

	ttlBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(ttlBytes, a.TTL)
	encoded = append(encoded, ttlBytes...)

	dataLengthBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(dataLengthBytes, uint16(len(a.Data)))
	encoded = append(encoded, dataLengthBytes...)

	encoded = append(encoded, a.Data...)

	return encoded
}
