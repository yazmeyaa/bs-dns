package header

import "encoding/binary"

const (
	RCODE_NO_ERROR uint8 = iota
	RCODE_FORMAT_ERROR
	RCODE_SERVER_FAILURE
	RCODE_NAME_ERROR
	RCODE_NOT_IMPLEMENTED
	RCODE_REFUSED
)

const (
	QR_REQUEST  bool = false
	QR_RESPONSE bool = true
)

type Header struct {
	ID                  uint16
	IsResponse          bool
	OPCODE              uint8
	AuthoritativeAnswer bool
	Truncation          bool
	RecursionAvailable  bool
	ResponseCode        uint8
	QDCount             uint16
	ANCount             uint16
	NSCount             uint16
	ARCount             uint16
}

func ReadHeader(data []byte) Header {
	header := Header{}

	header.ID = binary.BigEndian.Uint16(data[:2])

	flags := binary.BigEndian.Uint16(data[2:4])

	header.IsResponse = flags&(1<<15) != 0
	header.OPCODE = uint8((flags >> 11) & 0xF)
	header.AuthoritativeAnswer = flags&(1<<10) != 0
	header.Truncation = flags&(1<<9) != 0
	header.RecursionAvailable = flags&(1<<7) != 0
	header.ResponseCode = uint8(flags & 0xF)

	header.QDCount = binary.BigEndian.Uint16(data[4:6])
	header.ANCount = binary.BigEndian.Uint16(data[6:8])
	header.NSCount = binary.BigEndian.Uint16(data[8:10])
	header.ARCount = binary.BigEndian.Uint16(data[10:12])

	return header
}

func (h *Header) Encode() []byte {
	data := make([]byte, 12)

	binary.BigEndian.PutUint16(data[:2], h.ID)

	var flags uint16
	if h.IsResponse {
		flags |= 1 << 15
	}
	flags |= uint16(h.OPCODE&0xF) << 11
	if h.AuthoritativeAnswer {
		flags |= 1 << 10
	}
	if h.Truncation {
		flags |= 1 << 9
	}
	if h.RecursionAvailable {
		flags |= 1 << 7
	}
	flags |= uint16(h.ResponseCode & 0xF)

	binary.BigEndian.PutUint16(data[2:4], flags)

	binary.BigEndian.PutUint16(data[4:6], h.QDCount)
	binary.BigEndian.PutUint16(data[6:8], h.ANCount)
	binary.BigEndian.PutUint16(data[8:10], h.NSCount)
	binary.BigEndian.PutUint16(data[10:12], h.ARCount)

	return data
}
