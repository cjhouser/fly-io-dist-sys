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

	type broadcastRequestBody struct {
		Type     string              `json:"type"`
		Message  int                 `json:"message,omitempty"`
		Messages []int               `json:"messages,omitempty"`
		Topology map[string][]string `json:"topology,omitempty"`
	}

	type broadcastResponse struct {
		Type string `json:"type"`
	}

	type broadcastReadResponse struct {
		Type     string `json:"type"`
		Messages []int  `json:"messages"`
	}

	type broadcastTopologyResponse struct {
		Type string `json:"type"`
	}

	type broadcast struct {
		mutex          sync.Mutex
		neighbors      []string
		seen           map[int]struct{}
		unacknowledged *queue
	}

	bc := broadcast{
		neighbors:      []string{},
		seen:           map[int]struct{}{},
		unacknowledged: NewQueue(),
	}

	n.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		return nil
	})

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body broadcastRequestBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		bc.mutex.Lock()

		// I've never seen this message before
		if _, ok := bc.seen[body.Message]; !ok {
			bc.seen[body.Message] = struct{}{}

			for _, neighbor := range bc.neighbors {
				// I'll note the neighbors who need to ack this message
				bc.unacknowledged.Enqueue(
					&Unacknowledged{message: body.Message, neighbor: neighbor},
				)
				// Then I'll send the message
				n.Send(neighbor, body)
			}

			// I've got unacknowledged messages. Sending one again!
			unackd, err := bc.unacknowledged.Dequeue(nil)
			if err == nil {
				bc.unacknowledged.Enqueue(unackd)
				n.Send(unackd.neighbor, broadcastRequestBody{Type: "broadcast", Message: unackd.message})
			}
		}

		bc.mutex.Unlock()

		// Looks like sender never got my acknowledgement, I'll send another ack to them
		if _, err := bc.unacknowledged.Dequeue(
			&Unacknowledged{neighbor: msg.Src, message: body.Message},
		); err != nil {
			n.Send(msg.Src, broadcastRequestBody{Type: "broadcast", Message: body.Message})
		}

		responseBody := broadcastResponse{
			Type: "broadcast_ok",
		}

		return n.Reply(msg, responseBody)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var body broadcastRequestBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		bc.mutex.Lock()
		defer bc.mutex.Unlock()

		seenMessages := make([]int, len(bc.seen))
		i := 0
		for message := range bc.seen {
			seenMessages[i] = message
			i++
		}

		responseBody := broadcastReadResponse{
			Type:     "read_ok",
			Messages: seenMessages,
		}

		return n.Reply(msg, responseBody)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var body broadcastRequestBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		bc.mutex.Lock()
		defer bc.mutex.Unlock()

		bc.neighbors = append(bc.neighbors, body.Topology[n.ID()]...)

		responseBody := broadcastTopologyResponse{
			Type: "topology_ok",
		}

		return n.Reply(msg, responseBody)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
