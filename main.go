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

	type broadcastRequest struct {
		Type     string `json:"type"`
		Message  int    `json:"message"`
		Messages []int  `json:"messages"`
	}

	type broadcastReadResponse struct {
		Type     string `json:"type"`
		Messages []int  `json:"messages"`
	}

	type broadcastType struct {
		Type string `json:"type"`
	}

	type broadcastTopology struct {
		Type     string              `json:"type"`
		Topology map[string][]string `json:"topology"`
	}

	type broadcast struct {
		neighborAcks    map[string]map[int]struct{}
		neighborExpects map[string]map[int]struct{}
		mutex           sync.Mutex
		seens           map[int]struct{}
	}

	bc := broadcast{
		neighborExpects: map[string]map[int]struct{}{},
		neighborAcks:    map[string]map[int]struct{}{},
		seens:           map[int]struct{}{},
	}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		bc.mutex.Lock()
		defer bc.mutex.Unlock()

		var body broadcastRequest
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// Hacky way to handle messages from Maelstrom vs messages
		// from other workers
		if len(body.Messages) == 0 {
			body.Messages = append(body.Messages, body.Message)
			defer n.Reply(msg, broadcastType{Type: "broadcast_ok"})
		}

		for _, message := range body.Messages {
			if _, seen := bc.seens[message]; seen {
				if _, expected := bc.neighborExpects[msg.Src][message]; expected {
					delete(bc.neighborExpects[msg.Src], message)
				} else {
					// Recover from state where neighbor expects something that
					// I've already acknowledged
					bc.neighborAcks[msg.Src][message] = struct{}{}
				}
			} else {
				bc.seens[message] = struct{}{}
				// Another hack to keep Maelstrom nodes out of the data
				if _, isNode := bc.neighborAcks[msg.Src]; isNode {
					bc.neighborAcks[msg.Src][message] = struct{}{}
				}
				for neighbor := range bc.neighborExpects {
					if neighbor != msg.Src {
						bc.neighborExpects[neighbor][message] = struct{}{}
						for expect := range bc.neighborExpects[neighbor] {
							// little trick to dedupe. acks[neighbor] will be
							// wiped before the loop ends anyway
							bc.neighborAcks[neighbor][expect] = struct{}{}
						}
						outgoingMessages := make([]int, len(bc.neighborAcks[neighbor]))
						i := 0
						for outgoingMessage := range bc.neighborAcks[neighbor] {
							outgoingMessages[i] = outgoingMessage
							i++
						}
						n.Send(neighbor, broadcastRequest{
							Type:     "broadcast",
							Messages: outgoingMessages,
						})
						bc.neighborAcks[neighbor] = map[int]struct{}{}
					}
				}
			}
		}
		return nil
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		bc.mutex.Lock()
		defer bc.mutex.Unlock()

		var body broadcastType
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		seens := make([]int, len(bc.seens))
		i := 0
		for seen := range bc.seens {
			seens[i] = seen
			i++
		}

		return n.Reply(msg, broadcastReadResponse{
			Type:     "read_ok",
			Messages: seens,
		})
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		bc.mutex.Lock()
		defer bc.mutex.Unlock()

		var body broadcastTopology
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		topo := map[string][]string{}

		/*
			// Default topology
			topo = body.Topology[n.ID()]

		*/

		/*
			// Fully connected graph

			for i := 0; i < nodeCount; i++ {
				for j := 0; j < nodeCount; j++ {
					if i != j {
						topo[fmt.Sprintf("n%d", i)] = append(topo[fmt.Sprintf("n%d", i)], fmt.Sprintf("n%d", j))
					}
				}
			}
		*/

		/*
			// Highway
			nodeCount := 25
			for i := 1; i < nodeCount; i++ {
				topo[fmt.Sprintf("n%d", i-1)] = append(topo[fmt.Sprintf("n%d", i-1)], fmt.Sprintf("n%d", i))
				topo[fmt.Sprintf("n%d", i)] = append(topo[fmt.Sprintf("n%d", i)], fmt.Sprintf("n%d", i-1))
			}
		*/

		// Ring
		/*
			nodeCount := 25
			for i := 1; i < nodeCount; i++ {
				topo[fmt.Sprintf("n%d", i-1)] = append(topo[fmt.Sprintf("n%d", i-1)], fmt.Sprintf("n%d", i))
				topo[fmt.Sprintf("n%d", i)] = append(topo[fmt.Sprintf("n%d", i)], fmt.Sprintf("n%d", i-1))
			}
			topo["n0"] = append(topo["n0"], fmt.Sprintf("n%d", nodeCount-1))
			topo[fmt.Sprintf("n%d", nodeCount-1)] = append(topo[fmt.Sprintf("n%d", nodeCount-1)], "n0")

			// ring improvement: single connector
			topo["n0"] = append(topo["n0"], fmt.Sprintf("n%d", int(nodeCount/2)))
			topo[fmt.Sprintf("n%d", int(nodeCount/2))] = append(topo[fmt.Sprintf("n%d", int(nodeCount/2))], "n0")

			// ring improvement: double connector
			topo[fmt.Sprintf("n%d", int(nodeCount/4))] = append(topo[fmt.Sprintf("n%d", int(nodeCount/4))], fmt.Sprintf("n%d", int(nodeCount/4)+int(nodeCount/2)))
			topo[fmt.Sprintf("n%d", int(nodeCount/4)+int(nodeCount/2))] = append(topo[fmt.Sprintf("n%d", int(nodeCount/4)+int(nodeCount/2))], fmt.Sprintf("n%d", int(nodeCount/4)))
		*/

		// Hub-connected ring
		nodeCount := 25
		for i := 1; i < nodeCount; i++ {
			topo[fmt.Sprintf("n%d", i-1)] = append(topo[fmt.Sprintf("n%d", i-1)], fmt.Sprintf("n%d", i))
			topo[fmt.Sprintf("n%d", i)] = append(topo[fmt.Sprintf("n%d", i)], fmt.Sprintf("n%d", i-1))
		}

		topo["n0"] = append(topo["n0"], fmt.Sprintf("n%d", nodeCount-1))

		for i := 0; i < nodeCount; i = i + int((nodeCount)/6) - 1 {
			topo[fmt.Sprintf("n%d", nodeCount-1)] = append(topo[fmt.Sprintf("n%d", nodeCount-1)], fmt.Sprintf("n%d", i))
			topo[fmt.Sprintf("n%d", i)] = append(topo[fmt.Sprintf("n%d", i)], fmt.Sprintf("n%d", nodeCount-1))
		}

		/*
			// Recursive 5-node Cluster
			nodeCount := 25
			// Subclusters
			for i := 0; i < nodeCount; i = i + 5 {
				topo[fmt.Sprintf("n%d", i)] = []string{fmt.Sprintf("n%d", i+1), fmt.Sprintf("n%d", i+2), fmt.Sprintf("n%d", i+3), fmt.Sprintf("n%d", i+4)}
				topo[fmt.Sprintf("n%d", i+1)] = []string{fmt.Sprintf("n%d", i), fmt.Sprintf("n%d", i+2), fmt.Sprintf("n%d", i+4)}
				topo[fmt.Sprintf("n%d", i+2)] = []string{fmt.Sprintf("n%d", i), fmt.Sprintf("n%d", i+1), fmt.Sprintf("n%d", i+3)}
				topo[fmt.Sprintf("n%d", i+3)] = []string{fmt.Sprintf("n%d", i), fmt.Sprintf("n%d", i+2), fmt.Sprintf("n%d", i+4)}
				topo[fmt.Sprintf("n%d", i+4)] = []string{fmt.Sprintf("n%d", i), fmt.Sprintf("n%d", i+1), fmt.Sprintf("n%d", i+3)}
			}

			// Hub Cluster to Leaf Clusters
			topo["n1"] = append(topo["n1"], "n6")
			topo["n6"] = append(topo["n6"], "n1")
			topo["n2"] = append(topo["n2"], "n11")
			topo["n11"] = append(topo["n11"], "n2")
			topo["n3"] = append(topo["n3"], "n16")
			topo["n16"] = append(topo["n16"], "n3")
			topo["n4"] = append(topo["n4"], "n21")
			topo["n21"] = append(topo["n21"], "n4")

			// Leaf Cluster to Eachother
			topo["n7"] = append(topo["n7"], "n12")
			topo["n8"] = append(topo["n8"], "n22")
			topo["n12"] = append(topo["n12"], "n7")
			topo["n13"] = append(topo["n13"], "n17")
			topo["n17"] = append(topo["n17"], "n13")
			topo["n18"] = append(topo["n18"], "n23")
			topo["n22"] = append(topo["n22"], "n8")
			topo["n23"] = append(topo["n23"], "n18")
		*/

		for _, neighbor := range topo[n.ID()] {
			bc.neighborAcks[neighbor] = map[int]struct{}{}
			bc.neighborExpects[neighbor] = map[int]struct{}{}
		}

		return n.Reply(msg, broadcastType{
			Type: "topology_ok",
		})
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
