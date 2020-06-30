package utils

import (
	"bytes"
	"encoding/gob"
)

func MarshalGOB(in interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := gob.NewEncoder(buf).Encode(in)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalGOB(b []byte, in interface{}) error {
	buf := bytes.NewBuffer(b)
	return gob.NewDecoder(buf).Decode(in)
}
