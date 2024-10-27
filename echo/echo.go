package echo

import (
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Echo struct {
	Node *maelstrom.Node
}

func Init() *Echo {
	n := maelstrom.NewNode()

	e := Echo{
		Node: n,
	}

	n.Handle("echo", e.echo)

	return &e
}

func (e *Echo) echo(msg maelstrom.Message) error {
	// Unmarshal the message body as an loosely-typed map.
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	// Update the message type to return back.
	body["type"] = "echo_ok"

	// Echo the original message back with the updated message type.
	return e.Node.Reply(msg, body)
}
