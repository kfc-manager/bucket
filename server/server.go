package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/kfc-manager/bucket/domain"
)

type server struct {
	router  *http.ServeMux
	port    string
	auth    *domain.Auth
	storage *domain.Storage
}

func (s *server) middleware(methods map[string]http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if methods[r.Method] == nil {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "could not read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body)) // make the body re-readable

		headers := make(map[string]string)
		// go removes this header field for some reason from requests
		headers["host"] = r.Host
		for k := range r.Header {
			headers[strings.ToLower(k)] = r.Header.Get(k)
		}

		// for all S3 request this header must be present
		if len(headers["x-amz-content-sha256"]) < 1 {
			http.Error(w, "header x-amz-content-sha256 is missing", http.StatusBadRequest)
			return
		}
		bodyHash := domain.Sha256Hash(body)
		if headers["x-amz-content-sha256"] != bodyHash {
			http.Error(w, "content hash mismatch", http.StatusBadRequest)
			return
		}

		if err := s.auth.Validate(r.Method, r.RequestURI, headers, bodyHash); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// route to the correct handler for the method
		// (we checked at the start of the function if it exists)
		methods[r.Method].ServeHTTP(w, r)
	})
}

func New(port string, auth *domain.Auth, storage *domain.Storage) *server {
	s := &server{router: &http.ServeMux{}, port: port, auth: auth, storage: storage}
	routes := map[string]map[string]http.HandlerFunc{
		"/{name}": {
			"PUT": s.createBucket,
			// "GET": s.listBucket, TODO implement
		},
		"/{name}/{key}": {
			"GET":    s.getObject,
			"PUT":    s.putObject,
			"DELETE": s.deleteObject,
		},
	}
	s.router.HandleFunc("/", s.health)
	for path, route := range routes {
		s.router.Handle(path, s.middleware(route))
	}

	return s
}

func (s *server) Listen() error {
	return http.ListenAndServe(fmt.Sprintf(":%s", s.port), s.router)
}

func writeError(w http.ResponseWriter, err error) {
	if domErr, ok := err.(*domain.Error); ok {
		http.Error(w, err.Error(), domErr.Status)
	} else {
		log.Println("[ERROR] - " + err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("healthy"))
}

func (s *server) createBucket(w http.ResponseWriter, r *http.Request) {
	err := s.storage.NewBucket(r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(201)
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
	<CreateBucketConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
	</CreateBucketConfiguration>`))
}

func (s *server) getObject(w http.ResponseWriter, r *http.Request) {
	data, err := s.storage.Get(r.PathValue("name"), r.PathValue("key"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *server) putObject(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "could not read request body", http.StatusBadRequest)
	}
	defer r.Body.Close()

	if err := s.storage.Put(r.PathValue("name"), r.PathValue("key"), body); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("no content"))
}

func (s *server) deleteObject(w http.ResponseWriter, r *http.Request) {
	err := s.storage.Delete(r.PathValue("name"), r.PathValue("key"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("no content"))
}
