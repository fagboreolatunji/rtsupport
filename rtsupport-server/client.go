package main

import (
	"log"
	gocql "github.com/gocql/gocql"
	"github.com/gorilla/websocket"
)

type FindHandler func(string) (Handler, bool)

type Message struct {
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

type Client struct {
	send        	chan Message
	socket      	*websocket.Conn
	findHandler 	FindHandler
	session     	*gocql.Session
	stopChannels	map[int]chan bool
	userName		string
	userId			gocql.UUID
}

func (c *Client) NewStopChannel(stopKey int) chan bool {
	c.StopForKey(stopKey)
	stop := make(chan bool)
	c.stopChannels[stopKey] = stop
	return stop
}

func (c *Client) StopForKey(key int) {
	if ch, found := c.stopChannels[key]; found {
		ch <- true
		delete(c.stopChannels, key)
	}
}

func (client *Client) Read() {
	var message Message
	for {
		if err := client.socket.ReadJSON(&message); err != nil {
			break
		}
		if handler, found := client.findHandler(message.Name); found {
			handler(client, message.Data)
		}
	}
	client.socket.Close()
}

func (client *Client) Write() {
	for msg := range client.send {
		if err := client.socket.WriteJSON(msg); err != nil {
			break
		}
	}
	client.socket.Close()
}

func (c *Client) Close() {
	for _, ch := range c.stopChannels {
		ch <- true
	}
	close(c.send)
}

func NewClient(socket *websocket.Conn, findHandler FindHandler, session *gocql.Session) *Client {
	var user User
	user.Name = "anonymous"
	uuid, err := gocql.RandomUUID()
	if err != nil {
		log.Println(err.Error())
	}
	err = session.Query(`INSERT INTO user (id,name) VALUES (?, ?)`, uuid, user.Name).Exec()
	user.ID = uuid
	return &Client{
		send:        	make(chan Message),
		socket:      	socket,
		findHandler: 	findHandler,
		session:     	session,
		stopChannels:	make(map[int]chan bool),
		userName:		user.Name,
		userId:			uuid,
	}
}