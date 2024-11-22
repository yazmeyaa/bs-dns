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

	var clients sync.Map

	buffer := make([]byte, 2048)

	go func() {
		for {
			time.Sleep(10 * time.Second)
			clients.Range(func(key, value interface{}) bool {
				c := value.(*client)
				if time.Since(c.LastActive) > 30*time.Second {
					log.Printf("Удаление неактивного клиента: %s", key.(string))
					c.Conn.Close()
					clients.Delete(key)
				}
				return true
			})
		}
	}()

	for {
		n, clientAddr, err := localConn.ReadFrom(buffer)
		if err != nil {
			log.Printf("Ошибка чтения данных: %v", err)
			continue
		}

		log.Printf("Got message from client: %s", string(buffer[:n]))

		clientKey := clientAddr.String()

		clientValue, exists := clients.Load(clientKey)
		var c *client
		if !exists {
			remoteConn, err := net.Dial("udp", remoteAddr)
			if err != nil {
				log.Printf("Ошибка подключения к удалённому серверу для клиента %s: %v", clientKey, err)
				continue
			}
			c = &client{
				Addr:       clientAddr,
				LastActive: time.Now(),
				Conn:       remoteConn,
			}
			clients.Store(clientKey, c)

			go func(clientKey string, c *client) {
				remoteBuffer := make([]byte, 2048)
				for {
					n, err := c.Conn.Read(remoteBuffer)
					if err != nil {
						log.Printf("Ошибка чтения от удалённого сервера для клиента %s: %v", clientKey, err)
						clients.Delete(clientKey)
						return
					}

					_, err = localConn.WriteTo(remoteBuffer[:n], c.Addr)
					if err != nil {
						log.Printf("Ошибка отправки данных клиенту %s: %v", clientKey, err)
						return
					}
				}
			}(clientKey, c)
		} else {
			c = clientValue.(*client)
			c.LastActive = time.Now()
		}

		_, err = c.Conn.Write(buffer[:n])
		if err != nil {
			log.Printf("Ошибка пересылки данных от клиента %s на сервер: %v", clientKey, err)
		}
	}
}
