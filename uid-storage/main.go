package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Uid struct {
	Uid int `json:"uid"`
}

func uidClosure() func(http.ResponseWriter, *http.Request) {
	uids := map[int]struct{}{} // Empty structs don't use memory
	fmt.Printf("closure")

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var uid Uid
		err := json.NewDecoder(r.Body).Decode(&uid)
		if err != nil {
			fmt.Printf("failed to decode payload: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// collision check
		if _, ok := uids[uid.Uid]; ok {
			fmt.Printf("failed collision check: %v", err)
			http.Error(w, "conflict", http.StatusConflict)
			return
		}

		uids[uid.Uid] = struct{}{}
		fmt.Printf("saved uid %d\n", uid.Uid)
	}
}

func main() {
	fmt.Println("starting")
	http.HandleFunc("/uid", uidClosure())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
