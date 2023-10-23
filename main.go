package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/orange-v2/log"
	"github.com/reconquest/karma-go"
)

type bitfinexSubscribeMessage struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol,omitempty"`
	Key     string `json:"key,omitempty"`
}

func main() {
	symbols := getSymbols()
	log.Infoln("got symbols")

	url := "wss://api-pub.bitfinex.com/ws/2"

	i := 0
	var wg sync.WaitGroup
	done := make(chan bool)

	for i < len(symbols) {
		var errCtx *karma.Context
		conn, response, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			if response != nil {
				defer response.Body.Close()

				bodyBytes, err := io.ReadAll(response.Body)
				if err != nil {
					log.Errorx(err, "reading err response body")
				}

				errCtx = karma.
					Describe("code", response.Status).
					Describe("response_body", bodyBytes)
			}

			log.Errorx(errCtx.Reason(err), "websocket dial")
			return
		}
		defer conn.Close()

		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
				}

				_, msg, err := conn.ReadMessage()
				if err != nil {
					close(done)
					log.Errorx(err, "read message")
					return
				}

				if !strings.Contains(string(msg), "hb") {
					log.Infoln(string(msg))
				}
			}
		}()

		log.Infoln("successfully connected websocket")

		maxMessages := 30
		for j := 0; i < len(symbols) && j < maxMessages; i, j = i+1, j+1 {
			subscribeMessage := bitfinexSubscribeMessage{
				Event:   "subscribe",
				Channel: "trades",
				Symbol:  "t" + strings.ToUpper(symbols[i]),
			}

			msgBytes, err := json.Marshal(subscribeMessage)
			if err != nil {
				log.Errorx(err, "marshal message")
				return
			}

			err = conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				log.Errorx(err, "write subscribe message")
				return
			}
		}
	}

	wg.Wait()
}

func getSymbols() []string {
	symbols := make([]string, 0)

	// write code to get symbols through rest API from ""
	// and append them to symbols slice
	resp, err := http.Get("https://api.bitfinex.com/v1/symbols")
	if err != nil {
		log.Fatalx(err, "get symbols")
	}
	defer resp.Body.Close()

	// read response body line by line
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalx(err, "read resp body")
	}

	err = json.Unmarshal(bodyBytes, &symbols)
	if err != nil {
		log.Fatalx(err, "unmarshal symbols")
	}

	return symbols
}
