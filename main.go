package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Uid struct {
	Uid int `json:"uid"`
}

func main() {
	n := maelstrom.NewNode()

	// Echo
	n.Handle("echo", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// Update the message type to return back.
		body["type"] = "echo_ok"

		// Echo the original message back with the updated message type.
		return n.Reply(msg, body)
	})

	// Unique ID Generation
	n.Handle("generate", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		requestBody := Uid{}
		// retry new uid when storage detects a duplicate
		retry := true
		for retry {
			requestBody.Uid = rand.IntN(50000)

			requestBytes, err := json.Marshal(&requestBody)
			if err != nil {
				return err
			}

			reader := bytes.NewReader(requestBytes)

			resp, err := http.Post("http://uid-storage:8080/uid", "application/json", reader)
			if err != nil {
				return err
			}

			retry = resp.StatusCode == http.StatusConflict
			resp.Body.Close()
		}

		body["type"] = "generate_ok"
		body["id"] = requestBody.Uid

		return n.Reply(msg, body)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
