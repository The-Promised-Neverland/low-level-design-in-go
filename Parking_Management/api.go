package main

import (
	"encoding/json"
	"net/http"
)

type strategyRequest struct {
	Strategy string `json:"strategy"`
}

func newAPIHandler(aps *AutomatedParkingSystem, obs *observabilityStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, obs.buildState(aps.Snapshot()))
	})
	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, obs.Events())
	})
	mux.HandleFunc("POST /api/strategy", func(w http.ResponseWriter, r *http.Request) {
		var req strategyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		var strategy ParkingStrategy
		switch req.Strategy {
		case "NEAREST":
			strategy = NearestFloorStrategy{}
		case "LOAD_BALANCED":
			strategy = LoadBalancedStrategy{}
		case "ZONE_BALANCED":
			strategy = ZoneBalancedStrategy{}
		case "COMPACTION":
			strategy = CompactionStrategy{}
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": `strategy must be "NEAREST", "LOAD_BALANCED", "ZONE_BALANCED", or "COMPACTION"`,
			})
			return
		}
		if err := aps.SwitchStrategy(strategy); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"strategy": req.Strategy})
	})
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
