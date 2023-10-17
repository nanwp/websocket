package main

import (
	"log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type Message struct {
	User       User
	ReciptUser string
	Price      string
}

type User struct {
	Name     string
	Username string
}

type RegistrationInfor struct {
	Conn *websocket.Conn
	User User
}

func InitUsers() []User {
	users := make([]User, 0)
	user1 := User{
		Username: "nanda",
		Name:     "Nanda Wijaya Putra",
	}
	user2 := User{
		Username: "salman",
		Name:     "Salman Al",
	}
	user3 := User{
		Username: "rio",
		Name:     "Rio",
	}

	users = append(users, user1)
	users = append(users, user2)
	users = append(users, user3)

	return users
}

type hub struct {
	clients               map[*websocket.Conn]User
	clientRegisterChannel chan RegistrationInfor
	clientRemovalChannel  chan *websocket.Conn
	broadcastMessage      chan Message
}

func (h *hub) run() {
	for {
		select {
		case conn := <-h.clientRegisterChannel:
			h.clients[conn.Conn] = conn.User
		case conn := <-h.clientRemovalChannel:
			delete(h.clients, conn)
		case msg := <-h.broadcastMessage:
			for conn, user := range h.clients {
				if user.Username == msg.ReciptUser {
					_ = conn.WriteJSON(msg)
				}
			}
		}
	}
}

func main() {
	h := &hub{
		clients:               make(map[*websocket.Conn]User),
		clientRegisterChannel: make(chan RegistrationInfor),
		clientRemovalChannel:  make(chan *websocket.Conn),
		broadcastMessage:      make(chan Message),
	}

	users := InitUsers()

	go h.run()

	app := fiber.New()
	app.Use("/ws", AllowUpgrade)
	app.Use("/ws/bid", websocket.New(BidPrice(h, users)))
	_ = app.Listen(":8080")
}

func AllowUpgrade(ctx *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(ctx) {
		return ctx.Next()
	}
	return fiber.ErrUpgradeRequired
}

func BidPrice(h *hub, users []User) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		defer func() {
			log.Println("user disconnect", h.clients)
			h.clientRemovalChannel <- c
			_ = c.Close()
		}()

		username := c.Headers("Authorization")
		recipt := c.Query("recipt")

		userInfo := RegistrationInfor{
			Conn: c,
			User: User{
				Username: username,
			},
		}

		h.clientRegisterChannel <- userInfo

		log.Println("User connect", h.clients)

		for {
			messageType, price, err := c.ReadMessage()
			if err != nil {
				return
			}

			if messageType == websocket.TextMessage {
				h.broadcastMessage <- Message{
					User: User{
						Username: username,
					},
					ReciptUser: recipt,
					Price:      string(price),
				}
			}
		}
	}
}
