package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func ServeHTTP(s *Service, port int) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/datum/list_crds", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		res, err := s.ListCRDs(context.Background())
		if err != nil { http.Error(w, err.Error(), 500); return }
		writeJSON(w, res)
	})

	mux.HandleFunc("/datum/get_crd", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		var req GetCRDReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", 400); return
		}
		res, err := s.GetCRD(context.Background(), req)
		if err != nil { http.Error(w, err.Error(), 400); return }
		writeJSON(w, res)
	})

	mux.HandleFunc("/datum/validate_yaml", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		var req ValidateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", 400); return
		}
		writeJSON(w, s.ValidateYAML(context.Background(), req))
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return http.ListenAndServe(addr, mux)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
