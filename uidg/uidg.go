package uidg

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type UidGeneration struct {
	Node          *maelstrom.Node
	mutex         sync.Mutex
	sequence      int64
	lastTimestamp int64
}

func Init() *UidGeneration {
	n := maelstrom.NewNode()

	u := UidGeneration{
		Node:          n,
		sequence:      -1,
		lastTimestamp: time.Now().UnixMilli(),
	}

	n.Handle("generate", u.generate)

	return &u
}

func (u *UidGeneration) generate(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	timestamp := time.Now().UnixMilli()

	// Protect variables because of concurrent access
	u.mutex.Lock()
	if timestamp != u.lastTimestamp {
		// Reset the sequence number each millisecond to
		// refresh the pool of available UIDs and to keep
		// sequence consistent for each millisecond
		u.sequence = -1
	}

	u.sequence++
	u.lastTimestamp = timestamp
	u.mutex.Unlock()

	body["type"] = "generate_ok"
	body["id"] = fmt.Sprintf("%d%s%d", timestamp, u.Node.ID(), u.sequence)

	return u.Node.Reply(msg, body)
}
