package wshandlers

import (
	"log"
	"net/http"
	"time"

	"balance-web/internal/domain"
	infrastructure "balance-web/internal/infrastructure/websocket"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Prevent blocking cross-origin requests from iOS and Web clients during development
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handlers holds dependencies for WebSocket request operations.
type Handlers struct {
	hub *infrastructure.Hub
}

// NewHandlers creates a new WebSocket Handler injected with the Hub instance.
func NewHandlers(hub *infrastructure.Hub) *Handlers {
	return &Handlers{
		hub: hub,
	}
}

// RegisterRoutes links Echo pathways to handler closures.
func (h *Handlers) RegisterRoutes(e *echo.Echo) {
	e.GET("/ws", h.ServeWS)
}

// ServeWS initiates a WebSocket downgrade map from standard HTTP contexts.
func (h *Handlers) ServeWS(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println("Error upgrading to websocket:", err)
		return err
	}

	deviceType := c.Request().Header.Get("X-Client-Type")
	if deviceType == "" {
		deviceType = "web"
	}

	client := &infrastructure.Client{
		ID:         conn.RemoteAddr().String(),
		DeviceType: deviceType,
		Send:       make(chan *domain.WSEvent, 256),
	}
	
	h.hub.Register <- client

	// Start a simplistic read/write pump
	go h.writePump(client, conn)
	go h.readPump(client, conn)

	return nil
}

func (h *Handlers) readPump(client *infrastructure.Client, conn *websocket.Conn) {
	defer func() {
		h.hub.Unregister <- client
		conn.Close()
	}()
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
	}
}

func (h *Handlers) writePump(client *infrastructure.Client, conn *websocket.Conn) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteJSON(message); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
