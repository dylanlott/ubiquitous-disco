package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fly-apps/go-example/pkg/db"
)

// monitorHandler declares the whole monitor route
func (s *S) monitorHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := s.db.Find(&db.Monitor{})
		if result.Error != nil {
			w.Write([]byte(fmt.Sprintf("failed to get monitors from DB: %s", result.Error)))
			return
		}

		rows, err := result.Rows()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("failed to get monitor rows: %s", err)))
			return
		}

		var m *db.Monitor
		monitors := []*db.Monitor{}
		for rows.Next() {
			if err := rows.Scan(&m); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("failed to get monitor rows: %s", err)))
				return
			}
		}

		b, err := json.Marshal(monitors)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("failed to marshal json: %s", err)))
			return
		}

		w.WriteHeader(200)
		w.Write(b)
		return
	case http.MethodPost:
	case http.MethodPut:
	case http.MethodDelete:
	default:
		// TODO: handle crud for routes here
		w.WriteHeader(500)
		w.Write([]byte("not impl"))
	}
	return
}
