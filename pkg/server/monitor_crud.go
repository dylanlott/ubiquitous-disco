package server

import (
	"encoding/json"
	"net/http"

	"github.com/fly-apps/go-example/pkg/db"
	"github.com/gorilla/mux"
)

// monitorHandler declares the whole monitor route
func (s *S) monitorHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// define an empty list of monitors
		var monitors []*db.Monitor
		// find will mutate the monitors
		result := s.db.Find(&monitors)
		if result.Error != nil {
			http.Error(w, result.Error.Error(), http.StatusBadRequest)
			return
		}

		// encode the pointer to monitors
		json.NewEncoder(w).Encode(&monitors)
		return
	case http.MethodPost:
		var mon *db.Monitor
		if err := json.NewDecoder(r.Body).Decode(&mon); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// NB: Create function mutates `mon`
		tx := s.db.Create(&mon)
		if tx.Error != nil {
			http.Error(w, tx.Error.Error(), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(mon)
		return
	case http.MethodPut:
		vars := mux.Vars(r)
		if v, ok := vars["id"]; ok {
			var m *db.Monitor
			if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			tx := s.db.Save(&m)
			if tx.Error != nil {
				http.Error(w, tx.Error.Error(), http.StatusBadRequest)
				return
			}

			json.NewEncoder(w).Encode(&m)
			return
		}
		http.Error(w, "must provide id", http.StatusBadRequest)
		return

	case http.MethodDelete:
		vars := mux.Vars(r)
		if v, ok := vars["id"]; ok {
			tx := s.db.Delete(&db.Monitor{}, v)
			if tx.Error != nil {
				http.Error(w, tx.Error.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "must provide id", http.StatusBadRequest)
		return
	default:
		// TODO: handle crud for routes here
		w.WriteHeader(500)
		w.Write([]byte("not impl"))
	}
	return
}
