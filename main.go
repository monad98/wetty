package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/kr/pty"
)

//Message - JSON message struct
type Message struct {
	Type string `json:"type"`
	Msg  string `json:"msg"`
}

var addr = "localhost:11228"

func wsHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("bash")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Fatal("Error!")
	}

	defer func() {
		fmt.Println("Closed!!!")
		_ = ptmx.Close()
	}()

	//size
	initialSize := pty.Winsize{Rows: 30, Cols: 80, X: 15, Y: 15}
	pty.Setsize(ptmx, &initialSize)

	//ws
	upgrader := websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	// go readInput(c, readCh)
	// go writeOutput(ptmx, writeCh)
	go writeToWS(c, ptmx)
	readFromWS(c, ptmx)

}

func writeToWS(ws *websocket.Conn, r io.Reader) {
	buf := make([]byte, 128)
	for {
		n, err := r.Read(buf)
		if err != nil {
			log.Printf("Failed to read: %s", err)
			return
		}

		err = ws.WriteMessage(websocket.TextMessage, buf[0:n])

		if err != nil {
			log.Printf("Failed to send: %s", err)
			return
		}
	}
}

func readFromWS(ws *websocket.Conn, w *os.File) {
	for {
		var m Message
		if err := ws.ReadJSON(&m); err != nil {
			break
		}
		cmd := []byte(m.Msg)
		if m.Type == "input" {
			w.Write(cmd)
		} else if m.Type == "resize" {
			size := strings.Split(m.Msg, ":")
			col, _ := strconv.Atoi(size[1])
			row, _ := strconv.Atoi(size[0])
			pty.Setsize(w, &pty.Winsize{Rows: uint16(col), Cols: uint16(row), X:15, Y:15})
		}

	}
}

func main() {
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)
	http.HandleFunc("/terminal", wsHandler)
	log.Fatal(http.ListenAndServe(addr, nil))
}
