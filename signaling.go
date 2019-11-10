package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func signalingServer()  {
	upgrader := websocket.Upgrader{}

	connections := make(map[int]*websocket.Conn)

	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		socket, err := upgrader.Upgrade(writer, request, nil)
		if err != nil && err != io.EOF {
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
				if err == io.EOF {
					return
				}
				fmt.Println(err.Error())
				return
			}

			go func() {
				for {
					if len(connections) == 2 {
						for id, c := range connections {
							if id != cid {
								err = c.WriteMessage(websocket.TextMessage, msg)
								checkError(err)
							}
						}
						break;
					}
					time.Sleep(50 * time.Millisecond)
				}
			}()
		}
	})

	panic(http.ListenAndServe(":8443", nil))
}
