package records

import (
	"context"
	"errors"
	"net"

	"github.com/redis/go-redis/v9"
)

type DNSRecord struct {
	Label       string     `redis:"label"`
	Description string     `redis:"description"`
	Name        string     `redis:"name"`
	IPAddr      net.IPAddr `redis:"ipaddr"`
}

var (
	ErrRecordNotFound = errors.New("record not found")
)

func GetDNSRecord(ctx context.Context, rc *redis.Client, name string) (DNSRecord, error) {
	record := DNSRecord{}
	rc.HGet(ctx, name, "record")

	return record, nil
}
