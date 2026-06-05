package nps_mux

import (
	"encoding/binary"
	"errors"
	"github.com/astaxie/beego/logs"
	"io"
)

type basePackager struct {
	buf []byte
	// buf contain the mux protocol struct binary data, we copy data to buf firstly.
	// replace binary.Read/Write method, it may use reflect shows slowly.
	// also reduce conn.Read/Write calls which use syscall.
	// due to our test, conn.Write method reduce by two-thirds CPU times,
	// conn.Write method has 20% reduction of the CPU times,
	// totally provides more than twice of the CPU performance improvement.
	length  uint16
	content []byte
}

func (Self *basePackager) Set(content []byte) (err error) {
	Self.reset()
	if content != nil {
		n := len(content)
		//fmt.Println(content)
		if n == 0 {
			// 长度为0的包，不应该向上抛，不然客户端会EOF，这里暂时没解决空包的问题 TODO
			//logs.Error("mux:packer: newpack content is zero length")
			//err = errors.New("mux:packer: newpack content is zero length")
		}
		if n > maximumSegmentSize {
			logs.Error("mux:packer: newpack content segment too large")
			//err = errors.New("mux:packer: newpack content segment too large")
			return
		}
		Self.content = Self.content[:n]
		copy(Self.content, content)
	} else {
		logs.Error("mux:packer: newpack content is nil")
		return
		//panic("mux:packer: newpack content is nil")
		//err = errors.New("mux:packer: newpack content is nil")
	}
	Self.setLength()
	return
}

func (Self *basePackager) Pack(writer io.Writer) (err error) {
	binary.LittleEndian.PutUint16(Self.buf[5:7], Self.length)
	_, err = writer.Write(Self.buf[:7])
	if err != nil {
		return
	}
	_, err = writer.Write(Self.content[:Self.length])
	return
}

func (Self *basePackager) UnPack(reader io.Reader) (n uint16, err error) {
	Self.reset()
	l, err := io.ReadFull(reader, Self.buf[5:7])
	if err != nil {
		return
	}
	n += uint16(l)
	Self.length = binary.LittleEndian.Uint16(Self.buf[5:7])
	if int(Self.length) > cap(Self.content) {
		err = errors.New("mux:packer: unpack err, content length too large")
		return
	}
	if Self.length > maximumSegmentSize {
		err = errors.New("mux:packer: unpack content segment too large")
		return
	}
	Self.content = Self.content[:int(Self.length)]
	l, err = io.ReadFull(reader, Self.content)
	n += uint16(l)
	return
}

func (Self *basePackager) setLength() {
	Self.length = uint16(len(Self.content))
	return
}

func (Self *basePackager) reset() {
	Self.length = 0
	Self.content = Self.content[:0] // reset length
}

type muxPackager struct {
	flag   uint8
	id     int32
	window uint64
	basePackager
}

func (Self *muxPackager) Set(flag uint8, id int32, content interface{}) (err error) {
	Self.buf = windowBuff.Get()
	Self.flag = flag
	Self.id = id
	switch flag {
	case muxPingFlag, muxPingReturn, muxNewMsg, muxNewMsgPart:
		Self.content = windowBuff.Get()
		if content != nil {
			err = Self.basePackager.Set(content.([]byte))
		}
	case muxMsgSendOk:
		// MUX_MSG_SEND_OK contains one data
		Self.window = content.(uint64)
	}
	return
}

func (Self *muxPackager) Pack(writer io.Writer) (err error) {
	Self.buf = Self.buf[0:13]
	Self.buf[0] = byte(Self.flag)
	binary.LittleEndian.PutUint32(Self.buf[1:5], uint32(Self.id))
	switch Self.flag {
	case muxNewMsg, muxNewMsgPart, muxPingFlag, muxPingReturn:
		err = Self.basePackager.Pack(writer)
		windowBuff.Put(Self.content)
	case muxMsgSendOk:
		binary.LittleEndian.PutUint64(Self.buf[5:13], Self.window)
		_, err = writer.Write(Self.buf[:13])
	default:
		_, err = writer.Write(Self.buf[:5])
	}
	windowBuff.Put(Self.buf)
	return
}

func (Self *muxPackager) UnPack(reader io.Reader) (n uint16, err error) {
	Self.buf = windowBuff.Get()
	Self.buf = Self.buf[0:13]
	l, err := io.ReadFull(reader, Self.buf[:5])
	if err != nil {
		windowBuff.Put(Self.buf) // 归还 buf，避免 pool 泄漏
		return
	}
	n += uint16(l)
	Self.flag = uint8(Self.buf[0])
	Self.id = int32(binary.LittleEndian.Uint32(Self.buf[1:5]))
	switch Self.flag {
	case muxNewMsg, muxNewMsgPart, muxPingFlag, muxPingReturn:
		var m uint16
		Self.content = windowBuff.Get() // need Get a window buf from pool
		m, err = Self.basePackager.UnPack(reader)
		if err != nil {
			windowBuff.Put(Self.content) // 归还 content buf，避免 pool 泄漏
			windowBuff.Put(Self.buf)
			return
		}
		n += m
	case muxMsgSendOk:
		l, err = io.ReadFull(reader, Self.buf[5:13])
		Self.window = binary.LittleEndian.Uint64(Self.buf[5:13])
		n += uint16(l) // uint64
	}
	windowBuff.Put(Self.buf)
	return
}

func (Self *muxPackager) reset() {
	Self.id = 0
	Self.flag = 0
	Self.length = 0
	Self.content = nil
	Self.window = 0
	Self.buf = nil
}
