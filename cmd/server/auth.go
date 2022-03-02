package main

import (
	"flag"
	"net/http"
	"strings"
)

var (
	authUsername = flag.String("api-username", "admin", "Username for API endpoint")
	authPassword = flag.String("api-password", "", "Password for API endpoint")
	authToken    = flag.String("api-token", "", "Bearer token for authentication")
)

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		valid := true
		if *authPassword != "" {
			username, password, ok := r.BasicAuth()
			valid = ok && username == *authUsername && password == *authPassword
		} else if *authToken != "" {
			authHeader := r.Header.Get("Authentication")
			if authHeader != "" {
				tokens := strings.Split(authHeader, " ")
				switch tokens[0] {
				case "Bearer":
					valid = tokens[1] == *authToken
				}
			}
		}

		if valid {
			next.ServeHTTP(w, r)
		} else {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	})
}
