package main

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/orcaman/concurrent-map"
	"log"
	"net/http"
	"io/ioutil"
	"fmt"
	"gopkg.in/natefinch/lumberjack.v2"
)

type PostResponse struct {
	Digest string `json:"digest"`
}

type QueryResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	ErrorMessage string `json:"err_msg"`
}

type MyConcurrentMap struct {
	cMap *cmap.ConcurrentMap
}

func (m *MyConcurrentMap) secureHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]
	w.Header().Set("Content-Type", "application/json")
	if value, ok := m.cMap.Get(hash); ok {
		valueAsString := value.(string)
		response := &QueryResponse{Message: string(valueAsString)}
		f, _ := json.Marshal(response)
		w.Write([]byte(f))
		return
	}
	response := &ErrorResponse{ErrorMessage: "Message not found"}
	respondWithJSON(w, 404, response)
}

func (m *MyConcurrentMap) addHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		response := &ErrorResponse{ErrorMessage: "Message not found"}
		respondWithJSON(w, 404, response)
		return
	}
	payload := &QueryResponse{}
	err = json.Unmarshal(body, &payload)
	if err != nil {
		response := &ErrorResponse{ErrorMessage: "Message not found"}
		respondWithJSON(w, 404, response)
		return
	}
	h := sha256.New()
	h.Write([]byte(payload.Message))
	bar := h.Sum(nil)
	var sx16 string = fmt.Sprintf("%x", bar)
	m.cMap.Set(sx16, payload.Message)
	response := &PostResponse{Digest: sx16}
	respondWithJSON(w, 201, response)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	log.SetOutput(&lumberjack.Logger{
		Filename:   "/tmp/tls.log",
		MaxSize:    5, // megabytes
		MaxBackups: 3,
		MaxAge:     3, //days
	})
	// Concurrent HashMap
	bar := cmap.New()
	foo := &MyConcurrentMap{cMap: &bar}
	s := mux.NewRouter()
	s.HandleFunc("/messages", foo.addHandler).Methods("POST")
	s.HandleFunc("/messages/{hash}", foo.secureHandler).Methods("GET")

	log.Printf("Staring HTTPS service on %s .../n", ":443")
	if err := http.ListenAndServeTLS(":443", "localhost.crt", "server.key", s); err != nil {
		panic(err)
	}
}

