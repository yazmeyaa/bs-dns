package records

import (
	"net"
)

type DNSRecord struct {
	Label       string
	Description string
	Name        string
	IPAddr      string
}

func (r *DNSRecord) GetIPAddr() (*net.IPAddr, error) {
	return net.ResolveIPAddr("ip", r.IPAddr)
}

func (r *DNSRecord) GetIPAddrBytes() []byte {
	a, _ := r.GetIPAddr()
	return a.IP.To4()
}
