package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func checkAuth(w http.ResponseWriter, r *http.Request) bool {
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}

	return pair[0] == "shenal" && pair[1] == "admin"
}

func AuthHandler(w http.ResponseWriter, r *http.Request) {
	if checkAuth(w, r) {
		uploadFunc(w, r)
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
	w.WriteHeader(401)
	w.Write([]byte("401 Unauthorized\n"))
}
