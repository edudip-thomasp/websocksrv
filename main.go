package main

import (
        "net/http"
        "log"
        "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader {
        ReadBufferSize: 1024,
        WriteBufferSize: 1024,}

func homePage(w http.ResponseWriter, r *http.Request) {
    log.Printf("%v", w)
}

func reader(conn *websocket.Conn) {
    for {
        messageType, p, err := conn.ReadMessage()
        if err != nil {
            log.Fatal(err)
        }
        log.Println(string(p))

        err2 := conn.WriteMessage(messageType, p)

        if err2 != nil {
            log.Fatal(err)
        }
    }
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
    log.Printf("Websocket endpoint: respons: %v", w)
    upgrader.CheckOrigin = func(r *http.Request) bool {return true}

    ws, err := upgrader.Upgrade(w, r, nil)

    if err != nil {
        log.Fatal("Upgrading to websocket function failed")
    }

    log.Println("Successfully created websocket...")

    reader(ws)
}

func setupRoutes() {
        http.HandleFunc("/", homePage)
        http.HandleFunc("/ws", wsEndpoint)
}


func main() {
    log.Println("Go websockets")
    setupRoutes()
    log.Fatal(http.ListenAndServe(":8889", nil))
}
