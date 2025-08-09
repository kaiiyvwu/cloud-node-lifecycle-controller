package server

import (
	"cloud-node-lifecycle-controller/pkg/entity"
	"encoding/json"
	"net/http"
)

// NewAPIServer create new http server
func NewAPIServer(port string) {
	http.HandleFunc("/healthz", Healthz)
	http.ListenAndServe(":"+port, nil)
}

// Healthz health check api
func Healthz(w http.ResponseWriter, r *http.Request) {
	var res entity.HTTPResponse
	w.Header().Set("content-type", "application/json")
	switch r.Method {
	case "GET":
		res.Succ()
	}
	resp, _ := json.Marshal(res)
	w.Write(resp)
}
