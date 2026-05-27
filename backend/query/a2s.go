package query

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

type Info struct {
	Protocol    int
	Name        string
	Map         string
	Folder      string
	Game        string
	AppID       int
	Players     int
	MaxPlayers  int
	Bots        int
	ServerType  string
	Environment string
}

const challengePrefix = "\xff\xff\xff\xffTSource Engine Query\x00"

func QueryInfo(addr string, timeout time.Duration) (*Info, error) {
	conn, err := net.DialTimeout("udp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write([]byte(challengePrefix)); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	buf := make([]byte, 1400)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	resp := buf[:n]
	if n >= 9 && bytes.Equal(resp[:5], []byte("\xff\xff\xff\xffA")) {
		challenge := resp[5:9]
		req := append([]byte(challengePrefix), challenge...)
		if _, err := conn.Write(req); err != nil {
			return nil, fmt.Errorf("write2: %w", err)
		}
		n, err = conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("read2: %w", err)
		}
		resp = buf[:n]
	}
	return parseInfo(resp)
}

func parseInfo(data []byte) (*Info, error) {
	if len(data) < 6 {
		return nil, errors.New("short response")
	}
	if !bytes.Equal(data[:4], []byte{0xFF, 0xFF, 0xFF, 0xFF}) {
		return nil, errors.New("bad header")
	}
	if data[4] != 'I' {
		return nil, fmt.Errorf("unexpected response type %c", data[4])
	}
	r := bytes.NewReader(data[5:])
	var protocol uint8
	if err := binary.Read(r, binary.LittleEndian, &protocol); err != nil {
		return nil, err
	}
	name, err := readCString(r)
	if err != nil {
		return nil, err
	}
	mapName, err := readCString(r)
	if err != nil {
		return nil, err
	}
	folder, err := readCString(r)
	if err != nil {
		return nil, err
	}
	game, err := readCString(r)
	if err != nil {
		return nil, err
	}
	var appID uint16
	if err := binary.Read(r, binary.LittleEndian, &appID); err != nil {
		return nil, err
	}
	var players, maxPlayers, bots uint8
	if err := binary.Read(r, binary.LittleEndian, &players); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &maxPlayers); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &bots); err != nil {
		return nil, err
	}
	var serverType, env uint8
	if err := binary.Read(r, binary.LittleEndian, &serverType); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &env); err != nil {
		return nil, err
	}
	return &Info{
		Protocol:    int(protocol),
		Name:        name,
		Map:         mapName,
		Folder:      folder,
		Game:        game,
		AppID:       int(appID),
		Players:     int(players),
		MaxPlayers:  int(maxPlayers),
		Bots:        int(bots),
		ServerType:  string(rune(serverType)),
		Environment: string(rune(env)),
	}, nil
}

func readCString(r *bytes.Reader) (string, error) {
	var buf bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == 0 {
			return buf.String(), nil
		}
		buf.WriteByte(b)
	}
}
