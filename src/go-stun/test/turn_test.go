package test

import (
	"fmt"
	"github.com/pixelbender/go-stun/stun"
	"go-stun/turn"
	"testing"
)

func TestTurn(t *testing.T) {
	conn, err := turn.Allocate("turn:127.0.0.1:19302?transport=udp", "1624544580:sample", "+X3cBlaSOrGzmBGXcTUfKuBdFpU")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	fmt.Printf("Local address: %v, Relayed transport address: %v", conn.LocalAddr(), conn.RelayedAddr())
}

func TestSturn(t *testing.T) {
	conn, addr, err := stun.Discover("stun:stun.l.google.com:19302")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	fmt.Printf("Local address: %v, Server reflexive address: %v", conn.LocalAddr(), addr)
}
