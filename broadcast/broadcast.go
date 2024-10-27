package broadcast

import (
	"encoding/json"
	"fmt"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Broadcast struct {
	Node            *maelstrom.Node
	mutex           sync.Mutex
	seens           map[int]struct{}
	neighborAcks    map[string]map[int]struct{}
	neighborExpects map[string]map[int]struct{}
}

func Init() *Broadcast {
	n := maelstrom.NewNode()

	b := Broadcast{
		Node: n,
	}

	n.Handle("topology", b.topology)
	n.Handle("read", b.read)
	n.Handle("broadcast", b.broadcast)

	return &b
}

func (b *Broadcast) topology(msg maelstrom.Message) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	var body map[string]any
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
		// Sublusters
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

	for _, neighbor := range topo[b.Node.ID()] {
		b.neighborAcks[neighbor] = map[int]struct{}{}
		b.neighborExpects[neighbor] = map[int]struct{}{}
	}

	return b.Node.Reply(msg, map[string]any{
		"Type": "topology_ok",
	})
}

func (b *Broadcast) broadcast(msg maelstrom.Message) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	var body struct {
		Message  int
		Messages []int
	}
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	// Hacky way to handle messages from Maelstrom vs messages
	// from other workers
	if len(body.Messages) == 0 {
		body.Messages = append(body.Messages, body.Message)

		defer b.Node.Reply(msg, map[string]any{
			"Type": "broadcast_ok",
		})
	}

	for _, message := range body.Messages {
		if _, seen := b.seens[message]; seen {
			if _, expected := b.neighborExpects[msg.Src][message]; expected {
				delete(b.neighborExpects[msg.Src], message)
			} else {
				// Recover from state where neighbor expects something that
				// I've already acknowledged
				b.neighborAcks[msg.Src][message] = struct{}{}
			}
		} else {
			b.seens[message] = struct{}{}
			// Another hack to keep Maelstrom nodes out of the data
			if _, isNode := b.neighborAcks[msg.Src]; isNode {
				b.neighborAcks[msg.Src][message] = struct{}{}
			}
			for neighbor := range b.neighborExpects {
				if neighbor != msg.Src {
					b.neighborExpects[neighbor][message] = struct{}{}
					for expect := range b.neighborExpects[neighbor] {
						// little trick to dedupe. acks[neighbor] will be
						// wiped before the loop ends anyway
						b.neighborAcks[neighbor][expect] = struct{}{}
					}
					outgoingMessages := make([]int, len(b.neighborAcks[neighbor]))
					i := 0
					for outgoingMessage := range b.neighborAcks[neighbor] {
						outgoingMessages[i] = outgoingMessage
						i++
					}
					b.Node.Send(neighbor, map[string]any{
						"Type":     "broadcast",
						"Messages": outgoingMessages,
					})
					b.neighborAcks[neighbor] = map[int]struct{}{}
				}
			}
		}
	}
	return nil
}

func (b *Broadcast) read(msg maelstrom.Message) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	seens := make([]int, len(b.seens))
	i := 0
	for seen := range b.seens {
		seens[i] = seen
		i++
	}

	return b.Node.Reply(msg, map[string]any{
		"Type":     "read_ok",
		"Messages": seens,
	})
}
