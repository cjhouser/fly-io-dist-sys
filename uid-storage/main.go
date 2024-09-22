package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Uid struct {
	Uid int `json:"uid"`
}

func uidClosure() func(http.ResponseWriter, *http.Request) {
	sharedUidStorage := struct {
		mutex sync.Mutex
		uids  map[int]struct{} // Empty structs don't use memory
	}{
		uids: map[int]struct{}{},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var uid Uid
		err := json.NewDecoder(r.Body).Decode(&uid)
		if err != nil {
			fmt.Printf("failed to decode payload: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sharedUidStorage.mutex.Lock()
		_, collision := sharedUidStorage.uids[uid.Uid]
		sharedUidStorage.mutex.Unlock()

		if collision {
			fmt.Println("failed collision check")
			http.Error(w, "conflict", http.StatusConflict)
			return
		}

		sharedUidStorage.mutex.Lock()
		sharedUidStorage.uids[uid.Uid] = struct{}{}
		sharedUidStorage.mutex.Unlock()
	}
}

func main() {
	fmt.Println("starting")
	http.HandleFunc("/uid", uidClosure())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
