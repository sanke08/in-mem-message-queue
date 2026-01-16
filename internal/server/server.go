package server

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/sanke08/in-mem-message-queue/internal/broker"
	"github.com/sanke08/in-mem-message-queue/internal/utils"
)

type Server struct {
	ab  *broker.AuthenticatedBroker
	ctx context.Context
}

func NewServer(ctx context.Context, ab *broker.AuthenticatedBroker) *Server {
	return &Server{
		ab:  ab,
		ctx: ctx,
	}
}

func getAPIKey(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")

	scheme, key, found := strings.Cut(authHeader, " ")

	if !found || scheme != "ApiKey" {
		return "", false
	}

	key = strings.TrimSpace(key)

	if key == "" {
		return "", false
	}
	return key, true
}

func (s *Server) HandlePublish(w http.ResponseWriter, r *http.Request) {
	apiKey, ok := getAPIKey(r)
	if !ok {
		utils.JSONError(w, "missing api key", http.StatusUnauthorized)
		return
	}

	queueName := r.URL.Query().Get("queue")
	if queueName == "" {
		utils.JSONError(w, "queue name required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil || len(body) == 0 {
		utils.JSONError(w, "request body required", http.StatusBadRequest)
		return
	}

	msgID, err := s.ab.Publish(s.ctx, apiKey, queueName, body)

	if err != nil {
		utils.JSONError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	utils.JSONResponse(w, map[string]string{
		"message_id": msgID,
	}, http.StatusOK)
}

func (s *Server) HandleClaim(w http.ResponseWriter, r *http.Request) {
	apiKey, ok := getAPIKey(r)
	if !ok {
		utils.JSONError(w, "missing api key", http.StatusUnauthorized)
		return
	}
	queueName := r.URL.Query().Get("queue")
	if queueName == "" {
		utils.JSONError(w, "queue name required", http.StatusBadRequest)
		return
	}

	wait := 5
	msg, err := s.ab.Claim(s.ctx, apiKey, queueName, wait)
	if err != nil {
		utils.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}
	utils.JSONResponse(w, map[string]interface{}{
		"message_id": msg.ID,
		"payload":    string(msg.Payload),
	}, http.StatusOK)
}

func (s *Server) HandleAck(w http.ResponseWriter, r *http.Request) {
	apiKey, ok := getAPIKey(r)
	if !ok {
		utils.JSONError(w, "missing api key", http.StatusUnauthorized)
		return
	}
	queueName := r.URL.Query().Get("queue")
	messageID := r.URL.Query().Get("message_id")

	if queueName == "" || messageID == "" {
		utils.JSONError(w, "queue name and message_id required", http.StatusBadRequest)
		return
	}

	err := s.ab.Ack(s.ctx, apiKey, queueName, messageID)
	if err != nil {
		utils.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}
	utils.JSONResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	apiKey, ok := getAPIKey(r)
	if !ok {
		utils.JSONError(w, "missing api key", http.StatusUnauthorized)
		return
	}

	queueName := r.URL.Query().Get("queue")
	if queueName == "" {
		utils.JSONError(w, "queue name required", http.StatusBadRequest)
		return
	}

	stats, err := s.ab.Stats(s.ctx, apiKey, queueName)
	if err != nil {
		utils.JSONError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	utils.JSONResponse(w, stats, http.StatusOK)
}

func (s *Server) HandleCreateKey(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant")

	if tenantID == "" {
		utils.JSONError(w, "tenant ID required", http.StatusBadRequest)
		return
	}

	apiKey, err := s.ab.Auth.CreateKey(s.ctx, tenantID)
	if err != nil {
		utils.JSONError(w, "failed to create API key", http.StatusInternalServerError)
		return
	}

	utils.JSONResponse(w, map[string]string{
		"tenant":  tenantID,
		"api_key": apiKey,
	}, http.StatusOK)

}
