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
			log.Printf("Registering client: %v\n", client)
			clh.clients = append(clh.clients, client)
		case client := <-clh.unregister:
			for i, reggedClient := range clh.clients {
				if client == reggedClient {
					log.Printf("Client array BEFORE unregistering: %v\n", clh.clients)
					clh.clients = append(clh.clients[:i], clh.clients[i+1:]...)
					log.Printf("Client array AFTER unregistering: %v\n", clh.clients)
					break
				}
			}
		case cmm := <-clh.broadcast:
			for _, reggedClient := range clh.clients {
				log.Printf("Broadcastring to client: %v\n", reggedClient)
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

func reader(c *clientRepr, ch *clientHub) {
	defer func() {
		ch.unregister <- c
	    c.conn.Close()
	}()

	for {
		log.Printf("websocket reader(): At client: %v\n", c)
		_, p, err := c.conn.ReadMessage()

		if err != nil {
			log.Println("Error in websocket reader() detected: Maybe socket connection closed by client.")
			break
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
	defer func() {c.conn.Close()}()

	for {
		log.Printf("websocket writer(): At client: %v\n", c)

		select {
		case broadcastMsg := <-c.sendChan:
			writeErr := c.conn.WriteMessage(websocket.TextMessage, broadcastMsg)
			if writeErr != nil {
				log.Printf("Something went wrong when writing message to client. Closing thread...\n")
				log.Printf("Client: %+v\n error: %+v\n", c, writeErr)
				break
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

	hub.register <- client
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
