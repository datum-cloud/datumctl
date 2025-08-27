package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func ServeHTTP(s *Service, port int) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/datum/list_crds", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			jsonError(w, http.StatusUnauthorized, err)
			return
		}
		res, err := s.ListCRDs(r.Context())
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, res)
	})

	mux.HandleFunc("/datum/get_crd", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			jsonError(w, http.StatusUnauthorized, err)
			return
		}
		var req GetCRDReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		res, err := s.GetCRD(r.Context(), req)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, res)
	})

	mux.HandleFunc("/datum/validate_yaml", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			jsonError(w, http.StatusUnauthorized, err)
			return
		}
		var req ValidateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		// ValidateYAML returns a single value (the response struct)
		res := s.ValidateYAML(r.Context(), req)
		writeJSON(w, res)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return http.ListenAndServe(addr, mux)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
