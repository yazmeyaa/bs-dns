package main

import (
	"log"
	"net"
	"sync"
	"time"
)

type client struct {
	Addr       net.Addr
	LastActive time.Time
	Conn       net.Conn
}

func main() {
	localAddr := ":9339"
	remoteAddr := "game.brawlstarsgame.com:9339"

	localConn, err := net.ListenPacket("udp", localAddr)
	if err != nil {
		log.Fatalf("Ошибка создания UDP-сервера на %s: %v", localAddr, err)
	}
	defer localConn.Close()

	log.Printf("UDP-прокси запущен на %s, пересылает трафик в %s", localAddr, remoteAddr)

	clients := make(map[string]*client)
	mu := sync.Mutex{}

	buffer := make([]byte, 2048)

	go func() {
		for {
			time.Sleep(10 * time.Second)
			mu.Lock()
			for addr, c := range clients {
				if time.Since(c.LastActive) > 30*time.Second {
					log.Printf("Удаление неактивного клиента: %s", addr)
					c.Conn.Close()
					delete(clients, addr)
				}
			}
			mu.Unlock()
		}
	}()

	for {
		n, clientAddr, err := localConn.ReadFrom(buffer)
		if err != nil {
			log.Printf("Ошибка чтения данных: %v", err)
			continue
		}

		clientKey := clientAddr.String()

		mu.Lock()
		c, exists := clients[clientKey]
		if !exists {
			remoteConn, err := net.Dial("udp", remoteAddr)
			if err != nil {
				log.Printf("Ошибка подключения к удалённому серверу для клиента %s: %v", clientKey, err)
				mu.Unlock()
				continue
			}
			c = &client{
				Addr:       clientAddr,
				LastActive: time.Now(),
				Conn:       remoteConn,
			}
			clients[clientKey] = c

			go func(clientKey string, c *client) {
				remoteBuffer := make([]byte, 2048)
				for {
					n, err := c.Conn.Read(remoteBuffer)
					if err != nil {
						log.Printf("Ошибка чтения от удалённого сервера для клиента %s: %v", clientKey, err)
						mu.Lock()
						delete(clients, clientKey)
						mu.Unlock()
						return
					}

					mu.Lock()
					_, err = localConn.WriteTo(remoteBuffer[:n], c.Addr)
					mu.Unlock()
					if err != nil {
						log.Printf("Ошибка отправки данных клиенту %s: %v", clientKey, err)
						return
					}
				}
			}(clientKey, c)
		}
		c.LastActive = time.Now()
		mu.Unlock()

		_, err = c.Conn.Write(buffer[:n])
		if err != nil {
			log.Printf("Ошибка пересылки данных от клиента %s на сервер: %v", clientKey, err)
			continue
		}
	}
}
