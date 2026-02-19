package server

import (
	"net/http"
)

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.handleInfo)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("POST /ask", s.handleAsk)
	mux.HandleFunc("POST /task/notify", s.handleTaskStart)
	mux.HandleFunc("GET /stats", s.handleStats)

	return mux
}
