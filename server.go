package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
)

func server() {
	upgrader := websocket.Upgrader{}

	connections := make(map[int]*websocket.Conn)

	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		socket, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			log.Println(err)
			return
		}

		cid := rand.Int()
		connections[cid] = socket

		defer func() {
			delete(connections, cid)
		}()

		for {
			_, msg, err := socket.ReadMessage()
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			for id, c := range connections {
				if id != cid {
					err = c.WriteMessage(websocket.TextMessage, msg)
					checkError(err)
				}
			}
		}
	})

	panic(http.ListenAndServe(":8443", nil))
}
