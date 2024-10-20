package records

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
)

func FromMap(name string, hash map[string]string) (*DNSRecord, error) {
	label, ok := hash["Label"]
	if !ok {
		return nil, ErrWrongHash
	}

	description, ok := hash["Description"]
	if !ok {
		return nil, ErrWrongHash
	}
	ipAddr, ok := hash["IPAddr"]
	if !ok {
		return nil, ErrWrongHash
	}

	record := &DNSRecord{
		Name:        name,
		Label:       label,
		Description: description,
		IPAddr:      ipAddr,
	}

	return record, nil
}

func (r *DNSRecord) saveToRedis(ctx context.Context, rc *redis.Client) error {
	exist, err := rc.Exists(ctx, r.Name).Result()

	if err != nil {
		return err
	}

	if exist == 1 {
		return ErrRecordAlreadyExist
	}

	err = rc.HSet(ctx, r.Name, map[string]any{
		"Label":       r.Label,
		"Description": r.Description,
		"IPAddr":      r.IPAddr,
	}).Err()

	if err != nil {
		return err
	}

	return nil
}

func (r *DNSRecord) saveToDB(ctx context.Context, tx *sql.Tx) error {
	const query string = `INSERT INTO "dns_records" (label, description, name, ipaddr)
	VALUES($1, $2, $3, $4)`
	_, err := tx.ExecContext(ctx, query, r.Label, r.Description, r.Name, r.IPAddr)

	return err
}

func (r *DNSRecord) Save(ctx context.Context, rc *redis.Client, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	err = r.saveToDB(ctx, tx)

	if err != nil {
		tx.Rollback()
		return err
	}

	err = r.saveToRedis(ctx, rc)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func GetDNSRecord(ctx context.Context, rc *redis.Client, name string) (*DNSRecord, error) {
	hash, err := rc.HGetAll(ctx, name).Result()

	if err != nil {
		return nil, err
	}

	if len(hash) == 0 {
		return nil, ErrRecordNotFound
	}

	return FromMap(name, hash)
}
