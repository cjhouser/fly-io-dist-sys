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
		mutex               sync.Mutex
		neighbors           map[string]struct{}
		outstandingMessages map[int]map[string]struct{}
	}

	bc := broadcast{
		neighbors:           map[string]struct{}{},
		outstandingMessages: map[int]map[string]struct{}{},
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

		// I've never seen this message before, I should broadcast it
		if _, ok := bc.outstandingMessages[body.Message]; !ok {
			bc.outstandingMessages[body.Message] = map[string]struct{}{}
			for neighbor := range bc.neighbors {
				bc.outstandingMessages[body.Message][neighbor] = struct{}{}
			}

			// I've got messages that haven't been acknowledged. Sending them
			// now.
			for message, neighbors := range bc.outstandingMessages {
				bc := broadcastRequestBody{Type: "broadcast", Message: message}
				for neighbor := range neighbors {
					n.Send(neighbor, &bc)
				}
			}
		}
		// Looks like sender never got my acknowledgement, send it again
		if _, ok := bc.outstandingMessages[body.Message][msg.Src]; !ok {
			n.Send(msg.Src, broadcastRequestBody{Type: "broadcast", Message: body.Message})
		}

		// Handshake with sender is done
		delete(bc.outstandingMessages[body.Message], msg.Src)

		bc.mutex.Unlock()

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
		seenMessages := make([]int, len(bc.outstandingMessages))
		i := 0
		for message := range bc.outstandingMessages {
			seenMessages[i] = message
			i++
		}
		bc.mutex.Unlock()

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
		for _, neighbor := range body.Topology[n.ID()] {
			bc.neighbors[neighbor] = struct{}{}
		}
		bc.mutex.Unlock()

		responseBody := broadcastTopologyResponse{
			Type: "topology_ok",
		}

		return n.Reply(msg, responseBody)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
