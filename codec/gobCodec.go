package codec

import (
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	dec  *gob.Decoder
	enc  *gob.Encoder
}

func MakeGobCodecFunc(conn io.ReadWriteCloser) Codec {
	gobCodec := GobCodec{
		conn: conn,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(conn),
	}
	return &gobCodec
}
func (codec *GobCodec) ReadHeader(header *Header) error {
	if err := codec.dec.Decode(header); err != nil {
		log.Println("codec error:gob decoding header", err)
		return err
	}
	return nil
}
func (codec *GobCodec) ReadBody(body interface{}) error {
	if err := codec.dec.Decode(body); err != nil {
		log.Println("codec error:gob decoding body", err)
		return err
	}
	return nil
}
func (codec *GobCodec) WriteHeader(header Header) error {
	if err := codec.enc.Encode(header); err != nil {
		log.Println("codec error:gob encoding header", err)
		return err
	}
	return nil
}
func (codec *GobCodec) WriteBody(body interface{}) error {
	if err := codec.enc.Encode(body); err != nil {
		log.Println("codec error:gob encoding body", err)
		return err
	}
	return nil
}
func (codec *GobCodec) Close() error {
	return codec.conn.Close()
}
