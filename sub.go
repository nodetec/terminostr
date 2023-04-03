package main

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

const relayUrl string = "wss://relay.damus.io"
const npub string = "npub1qd3hhtge6vhwapp85q8eg03gea7ftuf9um4r8x4lh4xfy2trgvksf6dkva"
const limit int = 3

func sub() []*nostr.Event {
  var events []*nostr.Event 
	relay, err := nostr.RelayConnect(context.Background(), relayUrl)
	if err != nil {
		panic(err)
	}

	var filters nostr.Filters
	if _, v, err := nip19.Decode(npub); err == nil {
		pub := v.(string)
		filters = []nostr.Filter{{
			Kinds:   []int{30023},
			Authors: []string{pub},
			Limit:   limit,
		}}
	} else {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := relay.Subscribe(ctx, filters)

	go func() {
		<-sub.EndOfStoredEvents
		cancel()
	}()

	for ev := range sub.Events {
		// handle returned event.
		// channel will stay open until the ctx is cancelled (in this case, by calling cancel())
    events = append(events, ev)
	}

  return events
}
