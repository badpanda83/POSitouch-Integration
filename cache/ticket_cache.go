package cache

import (
	"encoding/json"
	"os"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

func WriteTicketsToCache(tickets []positouch.Ticket, path string) error {
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	return json.NewEncoder(f).Encode(tickets)
}

func ReadTicketsFromCache(path string) ([]positouch.Ticket, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var tickets []positouch.Ticket
	if err := json.NewDecoder(f).Decode(&tickets); err != nil { return nil, err }
	return tickets, nil
}