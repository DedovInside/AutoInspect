package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/notify"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func (h *AnalysisHandler) WSHandler(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := h.upgradeWebSocket(c)
	if err != nil {
		return
	}

	_, eventCh, unsubscribe := h.broadcastMgr.SubscribeUser(userID, 10)
	done := make(chan struct{})
	defer func() {
		unsubscribe()
		_ = conn.Close()
	}()

	go h.readWSMessages(conn, done, unsubscribe)
	h.writeWSEvents(c, conn, eventCh, done)
}

func (h *AnalysisHandler) upgradeWebSocket(c *gin.Context) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{CheckOrigin: h.checkWSOrigin}
	return upgrader.Upgrade(c.Writer, c.Request, nil)
}

func (h *AnalysisHandler) readWSMessages(conn *websocket.Conn, done chan<- struct{}, unsubscribe func()) {
	defer func() {
		close(done)
		unsubscribe()
		_ = conn.Close()
	}()
	if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *AnalysisHandler) writeWSEvents(c *gin.Context, conn *websocket.Conn, eventCh <-chan notify.JobEvent, done <-chan struct{}) {
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-done:
			return
		case <-pingTicker.C:
			if err := writeWSMessage(conn, websocket.PingMessage, nil); err != nil {
				return
			}
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			if err := writeWSJSON(conn, event); err != nil {
				return
			}
		}
	}
}

func writeWSMessage(conn *websocket.Conn, messageType int, data []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return conn.WriteMessage(messageType, data)
}

func writeWSJSON(conn *websocket.Conn, payload any) error {
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return conn.WriteJSON(payload)
}

func (h *AnalysisHandler) checkWSOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return false
	}
	return h.allowedOrigins[origin]
}
