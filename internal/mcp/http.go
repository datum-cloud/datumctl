package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func ServeHTTP(s *Service, port int) error {
	mux := http.NewServeMux()

	// ----- Phase-1 debug endpoints -----

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
		res := s.ValidateYAML(r.Context(), req)
		writeJSON(w, res)
	})

	// ----- debug endpoints (context + CRUD + list) -----

	mux.HandleFunc("/datum/change_context", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			jsonError(w, http.StatusUnauthorized, err)
			return
		}
		var req ChangeContextReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		res, err := s.ChangeContext(r.Context(), req)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, res)
	})

	mux.HandleFunc("/datum/create_resource", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			jsonError(w, http.StatusUnauthorized, err)
			return
		}
		var req CreateResourceReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		res, err := s.CreateResource(r.Context(), req)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, res)
	})

	mux.HandleFunc("/datum/get_resource", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			jsonError(w, http.StatusUnauthorized, err)
			return
		}
		var req GetResourceReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		res, err := s.GetResource(r.Context(), req)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, res)
	})

	mux.HandleFunc("/datum/update_resource", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			jsonError(w, http.StatusUnauthorized, err)
			return
		}
		var req UpdateResourceReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		res, err := s.UpdateResource(r.Context(), req)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, res)
	})

	mux.HandleFunc("/datum/delete_resource", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			jsonError(w, http.StatusUnauthorized, err)
			return
		}
		var req DeleteResourceReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		res, err := s.DeleteResource(r.Context(), req)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, res)
	})

	mux.HandleFunc("/datum/list_resources", func(w http.ResponseWriter, r *http.Request) {
		if err := s.K.Preflight(r.Context()); err != nil {
			jsonError(w, http.StatusUnauthorized, err)
			return
		}
		var req ListResourcesReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		res, err := s.ListResources(r.Context(), req)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
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
