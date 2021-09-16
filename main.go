package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type clientRepr struct {
	conn     *websocket.Conn
	sendChan chan []byte
}

type clientHub struct {
	register   chan *clientRepr
	unregister chan *clientRepr
	broadcast  chan *clientMsgMap
	clients    []*clientRepr
}

type clientMsgMap struct {
	client  *clientRepr
	message []byte
}

var hub clientHub

func (clh *clientHub) run() {
	for {
		select {
		case client := <-clh.register:
			clh.clients = append(clh.clients, client)
		case client := <-clh.unregister:
			for i, reggedClient := range clh.clients {
				if client == reggedClient {
					clh.clients = append(clh.clients[:i], clh.clients[i+1:]...)
					break
				}
			}
		case cmm := <-clh.broadcast:
			for _, reggedClient := range clh.clients {
				if cmm.client != reggedClient {
					reggedClient.sendChan <- cmm.message
				}
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v", w)
}

func reader(c *clientRepr, ch *clientHub) int {
	for {
		log.Printf("websocket reader(): At client: %v\n", c)
		_, p, err := c.conn.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}

		if len(p) > 0 {
			log.Printf("Server got message: %v\n", p)
			clientMsgMap := &clientMsgMap{
				client:  c,
				message: p,
			}
			ch.broadcast <- clientMsgMap
		}
		log.Println(string(p))
	}
}

func writer(c *clientRepr, ch *clientHub) {
	for {
		log.Printf("websocket writer(): At client: %v\n", c)

		select {
		case broadcastMsg := <-c.sendChan:
			err2 := c.conn.WriteMessage(websocket.TextMessage, broadcastMsg)
			if err2 != nil {
				log.Fatal(err2)
			}
		}
	}
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Printf("Websocket endpoint /ws hit: response|request: %+v|%+v\n", w, *r)
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		log.Fatal("Upgrading to websocket function failed")
	}

	log.Println("Successfully created websocket...")

	client := &clientRepr{
		conn:     ws,
		sendChan: make(chan []byte),
	}

	hub.clients = append(hub.clients, client)
	log.Printf("wsEndpoint(): Client registered just now: %v\n", client)

	go reader(client, &hub)
	go writer(client, &hub)
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

func main() {
	hub = clientHub{
		register:   make(chan *clientRepr),
		unregister: make(chan *clientRepr),
		broadcast:  make(chan *clientMsgMap),
		clients:    make([]*clientRepr, 0),
	}

	go hub.run()

	setupRoutes()
	log.Fatal(http.ListenAndServe(":8891", nil))
}
