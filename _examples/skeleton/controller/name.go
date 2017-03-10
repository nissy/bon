package controller

import (
	"net/http"
)

type Name struct {
	First string `toml:"first"`
	Last  string `toml:"last"`
}

func (n *Name) Init() error {
	return nil
}

func (n *Name) Hallo(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hallo," + n.Last + n.First))
}
