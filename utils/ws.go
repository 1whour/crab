package utils

import (
	"time"

	"github.com/gorilla/websocket"
)

func WriteMessageTimeout(conn *websocket.Conn, payload []byte, to time.Duration) error {
	conn.SetWriteDeadline(time.Now().Add(to))
	err := conn.WriteMessage(websocket.TextMessage, payload)
	conn.SetReadDeadline(time.Time{})
	return err
}

func WriteJsonTimeout(conn *websocket.Conn, x interface{}, to time.Duration) error {
	conn.SetWriteDeadline(time.Now().Add(to))
	err := websocket.WriteJSON(conn, x)
	conn.SetReadDeadline(time.Time{})
	return err
}
