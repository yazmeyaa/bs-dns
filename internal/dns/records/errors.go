package records

import "errors"

var (
	ErrRecordNotFound     = errors.New("record not found")
	ErrWrongHash          = errors.New("cannot decode hash")
	ErrRecordAlreadyExist = errors.New("record already exists")
)
