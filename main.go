package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const DiscoveryInterval = time.Hour

const DiscoveryServiceTag = "pubsub-chat-example"

func main() {
	nickFlag := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	roomFlag := flag.String("room", "awesome-chat-room", "name of chat room to join")
	flag.Parse()

	ctx := context.Background()

	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		panic(err)
	}

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	if err := setupDiscovery(h); err != nil {
		panic(err)
	}

	nick := *nickFlag
	if len(nick) == 0 {
		nick = defaultNick(h.ID())
	}

	room := *roomFlag

	cr, err := JoinChatRoom(ctx, ps, h.ID(), nick, room)
	if err != nil {
		panic(err)
	}

	ui := NewChatUI(cr)
	if err = ui.Run(); err != nil {
		printErr("error running text UI: %s", err)
	}
}

func printErr(m string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, m, args...)
}

func defaultNick(p peer.ID) string {
	return fmt.Sprintf("%s-%s", os.Getenv("USER"), shortID(p))
}

func shortID(p peer.ID) string {
	pretty := p.String()
	return pretty[len(pretty)-8:]
}

type discoveryNotifee struct {
	h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.ID)
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID, err)
	}
}

func setupDiscovery(h host.Host) error {
	s := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h})
	return s.Start()
}
