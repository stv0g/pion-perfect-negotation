package main

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/stv0g/pion-perfect-negotation/pkg"
)

type apiResponse struct {
	Sessions []pkg.Session `json:"sessions"`
}

func apiHandle(w http.ResponseWriter, r *http.Request) {
	resp := &apiResponse{}

	ss := []pkg.Session{}
	for name, s := range sessions {
		cs := []pkg.Connection{}
		for c := range s.Connections {
			cs = append(cs, c.Connection)
		}

		ss = append(ss, pkg.Session{
			Name:        name,
			Created:     s.Created,
			Connections: cs,
		})
	}

	resp.Sessions = ss

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logrus.Errorf("Failed to encode API response: %s", err)
	}
}
