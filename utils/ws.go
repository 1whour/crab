package utils

import (
	"time"

	"github.com/gorilla/websocket"
)

func WriteJsonTimeout(conn *websocket.Conn, x interface{}, to time.Duration) error {
	conn.SetWriteDeadline(time.Now().Add(to))
	err := websocket.WriteJSON(conn, x)
	conn.SetReadDeadline(time.Time{})
	return err
}
