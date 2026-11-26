package web

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// upgrader configures the HTTP -> websocket upgrade. CheckOrigin allows any
// origin: this is a local learning tool, not a multi-tenant service.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// wsConn adapts a gorilla/websocket connection to the conn interface used by
// Session. All writes go through a single writer goroutine fed by sendCh so
// concurrent flow output never interleaves into partial frames.
type wsConn struct {
	c      *websocket.Conn
	sendCh chan wsMessage
	close  func() error
}

const (
	writeWait = 10 * time.Second
	pongWait  = 60 * time.Second
	pingItvl  = 30 * time.Second
)

func newWSConn(c *websocket.Conn) *wsConn {
	w := &wsConn{c: c, sendCh: make(chan wsMessage, 64), close: c.Close}
	go w.writeLoop()
	return w
}

func (w *wsConn) read() (wsMessage, bool) {
	var m wsMessage
	if err := w.c.ReadJSON(&m); err != nil {
		return wsMessage{}, false
	}
	return m, true
}

func (w *wsConn) send(m wsMessage) error {
	select {
	case w.sendCh <- m:
		return nil
	default:
		// The browser stopped reading; drop the frame instead of blocking
		// the session goroutine forever.
		return websocket.ErrCloseSent
	}
}

func (w *wsConn) closeWrite() error { return w.close() }

func (w *wsConn) writeLoop() {
	ticker := time.NewTicker(pingItvl)
	defer func() {
		ticker.Stop()
		_ = w.c.Close()
	}()
	for {
		select {
		case msg, ok := <-w.sendCh:
			if !ok {
				return
			}
			_ = w.c.SetWriteDeadline(time.Now().Add(writeWait))
			if err := w.c.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = w.c.SetWriteDeadline(time.Now().Add(writeWait))
			if err := w.c.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Server is the web front-end of the application. Create it with NewServer
// and stop it with Shutdown.
type Server struct {
	Mux    *http.ServeMux
	hub    *hub
	shut   sync.Once
	server *http.Server
}

// NewServer builds the HTTP handler tree: the page itself at "/" (served
// from the embedded static assets) and the live session at "/ws".
func NewServer() *Server {
	s := &Server{hub: newHub()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.Handle("/static/", StaticHandler("/static/"))
	mux.HandleFunc("/ws", s.handleWS)
	s.Mux = mux
	return s
}

// ServeHTTP lets *Server be passed directly to http.Server / httptest.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Mux.ServeHTTP(w, r)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	ServeIndex(w, r)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("web: upgrade failed: %v", err)
		return
	}
	conn := newWSConn(c)

	// Keep the connection alive while the session runs; the standard
	// deadline/pong machinery detects dead browsers.
	c.SetReadLimit(4096)
	_ = c.SetReadDeadline(time.Now().Add(pongWait))
	c.SetPongHandler(func(string) error {
		return c.SetReadDeadline(time.Now().Add(pongWait))
	})

	session := NewSession(conn, s.hub)
	go session.Serve()

	// Pump keystrokes from the browser into the session until the socket
	// dies or the session terminates.
	for {
		m, ok := conn.read()
		if !ok {
			break
		}
		if m.Type == "key" {
			if !session.DeliverKey(m) {
				break
			}
		}
		_ = c.SetReadDeadline(time.Now().Add(pongWait))
	}
	session.Close()
}

// Shutdown closes every live session (so their goroutines unwind) and stops
// accepting connections. Safe to call more than once.
func (s *Server) Shutdown() {
	s.shut.Do(func() {
		s.hub.closeAll()
		if s.server != nil {
			_ = s.server.Close()
		}
	})
}

// ListenAndServe starts the server on addr and records it for Shutdown.
func (s *Server) ListenAndServe(addr string) error {
	srv := &http.Server{Addr: addr, Handler: s}
	s.server = srv
	err := srv.ListenAndServe()
	s.Shutdown()
	return err
}
