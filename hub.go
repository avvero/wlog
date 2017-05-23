// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"sync"
	"github.com/avvero/stomp/frame"
	"bytes"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	subscriptions      map[string]map[string]*Subscription
	subscriptionsMutex sync.Mutex

	// Inbound messages from the clients.
	broadcast chan *frame.Frame

	// Register requests from the clients.
	register chan *Subscription

	// Unregister requests from clients.
	unregister chan *Subscription
}

func newHub() *Hub {
	return &Hub{
		broadcast:     make(chan *frame.Frame),
		register:      make(chan *Subscription),
		unregister:    make(chan *Subscription),
		subscriptions: make(map[string]map[string]*Subscription),
	}
}

func (h *Hub) subscribe(subscription *Subscription) {
	h.subscriptionsMutex.Lock()
	defer h.subscriptionsMutex.Unlock()

	log.Printf("subscribe client to : %v", subscription)

	subscriptions, ok := h.subscriptions[subscription.destination]
	if ok == false {
		subscriptions = make(map[string]*Subscription)
		h.subscriptions[subscription.destination] = subscriptions
	}
	id := (*subscription.session).ID()
	if _, ok = subscriptions[id]; ok == false {
		subscriptions[id] = subscription
		go subscription.doSend()
	}
}

func (h *Hub) registerMarket(destination string) {
	h.subscriptionsMutex.Lock()
	defer h.subscriptionsMutex.Unlock()

	log.Printf("new destination : %v", destination)
	if h.subscriptions[destination] == nil {
		subscriptions := make(map[string]*Subscription)
		h.subscriptions[destination] = subscriptions
	}
}

func (h *Hub) unsubscribe(subscription *Subscription) {
	h.subscriptionsMutex.Lock()
	defer h.subscriptionsMutex.Unlock()

	log.Printf("unsubscribe client on : %v", subscription)

	if subscriptions, ok := h.subscriptions[subscription.destination]; ok == true {
		id := (*subscription.session).ID()
		if _, ok = subscriptions[id]; ok == true {
			delete(subscriptions, id)
			close(subscription.send)
		}
	}
}

func (h *Hub) run() {
	for {
		select {
		case subscription := <-h.register:
			h.subscribe(subscription)
		case subscription := <-h.unregister:
			h.unsubscribe(subscription)
		case fr := <-h.broadcast:
			destination := fr.Header.Get("destination")
			if h.subscriptions[destination] == nil {
				h.registerMarket(destination)
			}
			//log.Printf("v.send <- h.broadcast | to " + destination)
			//log.Printf("broadcasting on %s the %s for clients %d", destination, frame, len(h.subscriptions))

			//fr.Header.Add("subscription", subscription.id)
			fr.Header.Add("subscription", "sub-0")
			buf := bytes.NewBufferString("")
			frame.NewWriter(buf).Write(fr)
			frameString := buf.String()

			for _, subscription := range h.subscriptions[destination] {
				subscription.send <- frameString
			}
		}
	}
}
