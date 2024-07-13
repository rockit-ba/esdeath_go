package raftkv

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
)

type Server struct {
	addr   string
	ln     net.Listener
	raftKV *RaftKV
}

func NewJoinProcessor(addr string, raftKV *RaftKV) *Server {
	return &Server{
		addr:   addr,
		raftKV: raftKV,
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Error("Failed to start listener", "addr", s.addr, "err", err)
		return err
	}
	s.ln = ln

	http.HandleFunc("/join", s.handleJoin)
	http.HandleFunc("/remove", s.handleRemove)

	server := &http.Server{Addr: s.addr}
	go func() {
		if err := server.Serve(s.ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server start failed", "err", err)
			os.Exit(1)
		}
	}()

	log.Info("HTTP server started", "addr", s.addr)
	return nil
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var m map[string]string
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if len(m) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	remoteAddr, addrOk := m["addr"]
	nodeID, idOk := m["id"]
	if !addrOk || !idOk {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := s.raftKV.join(nodeID, remoteAddr); err != nil {
		log.Error("Failed to join node", "nodeID", nodeID, "remoteAddr", remoteAddr, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	// todo
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
