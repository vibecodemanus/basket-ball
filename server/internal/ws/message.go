package ws

import (
	"bytes"
	"encoding/json"
	"sync"
)

// Client -> Server message types
const (
	MsgPlayerInput uint8 = 0x01
	MsgJoinQueue   uint8 = 0x02
	MsgPing        uint8 = 0x04
)

// Server -> Client message types
const (
	MsgGameState          uint8 = 0x81
	MsgGameStart          uint8 = 0x82
	MsgGameOver           uint8 = 0x83
	MsgScored             uint8 = 0x84
	MsgPong               uint8 = 0x86
	MsgPlayerDisconnected uint8 = 0x87
	MsgTournamentResult   uint8 = 0x88
)

type Message struct {
	Type    uint8           `json:"type"`
	Tick    uint32          `json:"tick"`
	Payload json.RawMessage `json:"payload"`
}

type PlayerInputPayload struct {
	MoveX int8 `json:"moveX"`
	Jump  bool `json:"jump"`
	Shoot bool `json:"shoot"`
}

type JoinQueuePayload struct {
	Name string `json:"name"`
}

type PingPayload struct {
	ClientTime uint64 `json:"clientTime"`
}

type GameStartPayload struct {
	PlayerIndex  uint8     `json:"playerIndex"`
	Names        [2]string `json:"names"`
	IsTournament bool      `json:"isTournament,omitempty"`
}

type TournamentPlayerStats struct {
	Nickname    string `json:"nickname"`
	Wins        int    `json:"wins"`
	Losses      int    `json:"losses"`
	Draws       int    `json:"draws"`
	PointsFor   int    `json:"pointsFor"`
	GamesPlayed int    `json:"gamesPlayed"`
}

type TournamentResultPayload struct {
	YourStats     TournamentPlayerStats `json:"yourStats"`
	OpponentStats TournamentPlayerStats `json:"opponentStats"`
}

type PongPayload struct {
	ClientTime uint64 `json:"clientTime"`
	ServerTime uint64 `json:"serverTime"`
}

type PlayerDisconnectedPayload struct {
	PlayerIndex uint8 `json:"playerIndex"`
}

// bufPool recycles encoding buffers to reduce GC pressure in the hot path.
// At 60 Hz × 100 rooms, this avoids ~12 000 alloc/s from json.Marshal.
var bufPool = sync.Pool{
	New: func() any { return bytes.NewBuffer(make([]byte, 0, 512)) },
}

// Encode serializes a Message to JSON using a pooled buffer.
func Encode(msg Message) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	err := json.NewEncoder(buf).Encode(msg)
	if err != nil {
		bufPool.Put(buf)
		return nil, err
	}
	// json.Encoder.Encode appends '\n' — trim it for clean WS messages.
	raw := buf.Bytes()
	if len(raw) > 0 && raw[len(raw)-1] == '\n' {
		raw = raw[:len(raw)-1]
	}
	// Copy out so the buffer can be returned to the pool immediately.
	out := make([]byte, len(raw))
	copy(out, raw)
	bufPool.Put(buf)
	return out, nil
}

func Decode(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}

// NewMessage constructs a Message with a JSON-encoded payload (pooled buffer).
func NewMessage(typ uint8, tick uint32, payload any) (Message, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	err := json.NewEncoder(buf).Encode(payload)
	if err != nil {
		bufPool.Put(buf)
		return Message{}, err
	}
	raw := buf.Bytes()
	if len(raw) > 0 && raw[len(raw)-1] == '\n' {
		raw = raw[:len(raw)-1]
	}
	data := make([]byte, len(raw))
	copy(data, raw)
	bufPool.Put(buf)
	return Message{
		Type:    typ,
		Tick:    tick,
		Payload: json.RawMessage(data),
	}, nil
}
