package goc

import (
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type GrowOnlyCounter struct {
	Node *maelstrom.Node
	kv   *maelstrom.KV
}

func Init() *GrowOnlyCounter {
	n := maelstrom.NewNode()
	kv := maelstrom.NewSeqKV(n)
	g := GrowOnlyCounter{
		Node: n,
		kv:   kv,
	}

	n.Handle("add", g.add)
	n.Handle("read", g.read)

	return &g
}

func (g *GrowOnlyCounter) add(msg maelstrom.Message) error {
	// Unmarshal the message body as an loosely-typed map.
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	// Update the message type to return back.
	ret := map[string]any{
		"type": "add_ok",
	}

	return g.Node.Reply(msg, ret)
}

func (g *GrowOnlyCounter) read(msg maelstrom.Message) error {
	// Unmarshal the message body as an loosely-typed map.
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	// Update the message type to return back.
	ret := map[string]any{
		"type":  "read_ok",
		"value": 1234,
	}

	return g.Node.Reply(msg, ret)
}
