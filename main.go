package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

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
	var uidMutex sync.Mutex
	uidSequence := -1
	uidLastTimestamp := time.Now().UnixMilli()
	n.Handle("generate", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		timestamp := time.Now().UnixMilli()

		// Protect variables because of concurrent access
		uidMutex.Lock()
		if timestamp != uidLastTimestamp {
			// Reset the sequence number each millisecond to
			// refresh the pool of available UIDs and to keep
			// sequence consistent for each millisecond
			uidSequence = -1
		}

		uidSequence++
		uidLastTimestamp = timestamp
		uidMutex.Unlock()

		body["type"] = "generate_ok"
		body["id"] = fmt.Sprintf("%d%s%d", timestamp, n.ID(), uidSequence)

		return n.Reply(msg, body)
	})

	// Broadcast
	type broadcastBody struct {
		Type     string              `json:"type"`
		Message  int                 `json:"message,omitempty"`
		Messages []int               `json:"messages,omitempty"`
		Topology map[string][]string `json:"topology,omitempty"`
	}

	// Use a "set" for quick lookups
	var broadcastMutex sync.Mutex
	var broadcastNeighbors []string
	broadcastSeen := map[int]struct{}{} // empty structs use no memory

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body broadcastBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		broadcastMutex.Lock()
		if _, seen := broadcastSeen[body.Message]; !seen {
			broadcastSeen[body.Message] = struct{}{}
		}
		broadcastMutex.Unlock()

		for _, neighbor := range broadcastNeighbors {
			n.Send(neighbor, body)
		}

		responseBody := broadcastBody{
			Type: "broadcast_ok",
		}

		return n.Reply(msg, responseBody)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var body broadcastBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		broadcastMutex.Lock()
		seenMessages := make([]int, len(broadcastSeen))
		i := 0
		for message := range broadcastSeen {
			seenMessages[i] = message
			i++
		}
		broadcastMutex.Unlock()

		responseBody := broadcastBody{
			Type:     "read_ok",
			Messages: seenMessages,
		}

		return n.Reply(msg, responseBody)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var body broadcastBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		broadcastMutex.Lock()
		broadcastNeighbors = body.Topology[n.ID()]
		broadcastMutex.Unlock()

		responseBody := broadcastBody{
			Type: "topology_ok",
		}

		return n.Reply(msg, responseBody)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
