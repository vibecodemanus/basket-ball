package ws

import "encoding/json"

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
	PlayerIndex uint8 `json:"playerIndex"`
}

type PongPayload struct {
	ClientTime uint64 `json:"clientTime"`
	ServerTime uint64 `json:"serverTime"`
}

type PlayerDisconnectedPayload struct {
	PlayerIndex uint8 `json:"playerIndex"`
}

func Encode(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

func Decode(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}

func NewMessage(typ uint8, tick uint32, payload any) (Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Type:    typ,
		Tick:    tick,
		Payload: json.RawMessage(data),
	}, nil
}
