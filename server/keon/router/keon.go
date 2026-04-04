package router

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/zxdev/kvs"
)

type KEONServer struct {
	keon   *kvs.KEON
	insert func([]byte) struct{ Ok, Exist, NoSpace bool }
	lookup func(key []byte) (ok bool)
	remove func([]byte) struct{ Ok, Exist bool }
	sync.RWMutex
}

// create handler deafults to 1mm object container default
//
//	.../create/{size [1m 10m 100m {n} drop|reset ]}
func (kn *KEONServer) CreateHandler() http.HandlerFunc {

	type Response struct {
		Message string `json:"message,omitempty"`
		N       uint64 `json:"n,omitempty"`
	}

	var size atomic.Uint64
	return func(w http.ResponseWriter, r *http.Request) {

		var resp = Response{Message: "create"}
		var v = filepath.Base(r.URL.Path)
		switch v {
		case "drop", "0":
			resp.Message = "drop"

		case "reset":
			resp.Message = "reset"
			resp.N = size.Load()

		case "1m":
			resp.N = 1000000
		case "10m":
			resp.N = 10000000
		case "100m":
			resp.N = 100000000

		default:
			var c byte
			for _, c = range []byte(v) {
				if c > 47 && c < 58 {
					resp.N = resp.N*10 + uint64(c-48)
				} else {
					resp.N = 0
					break
				}
			}
		}

		if resp.N == 0 {
			kn.Lock()
			kn.keon = nil
			kn.insert = nil
			kn.remove = nil
			kn.Unlock()
			resp.Message = "drop"

		} else {
			kn.Lock()
			kn.keon = kvs.NewKEON(int(resp.N), nil)
			kn.insert = kn.keon.Insert(true)
			kn.lookup = kn.keon.Lookup()
			kn.remove = kn.keon.Remove()
			kn.Unlock()

			size.Store(resp.N)
			resp.Message = "create"

		}

		log.Println("manage:", resp.Message, resp.N)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&resp)

	}
}

// Status returns the current count, max size, and percent utilization
//
// .../status
func (kn *KEONServer) StatusHandler() http.HandlerFunc {

	type Stats struct {
		Count   int `json:"count,omitempty"`
		Max     int `json:"max,omitempty"`
		Percent int `json:"percent,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		if kn.keon == nil {
			w.WriteHeader(http.StatusFailedDependency) // 424

		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(&Stats{
				Count:   int(kn.keon.Len()),
				Max:     int(kn.keon.Cap()),
				Percent: int(kn.keon.Ratio()),
			})
		}
	}
}

// insert item
//
//	503 unavailable
//	400 insert failed
//	200 success
//
//	.../insert/{key}
func (kn *KEONServer) InsertHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		kn.Lock()
		defer kn.Unlock()

		if kn.insert == nil {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			return
		}

		switch r.Method {
		case "GET":

			if !kn.insert([]byte(filepath.Base(r.URL.Path))).Ok {
				w.WriteHeader(http.StatusBadRequest) // 400
				return
			}

		case "POST":
			scanner := bufio.NewScanner(r.Body)
			for scanner.Scan() {
				if !kn.insert(scanner.Bytes()).Ok {
					w.WriteHeader(http.StatusBadRequest) // 400
					return
				}
			}

		}

		w.WriteHeader(http.StatusOK) // 200
	}
}

// remove item
//
//	503 unavailable
//	200 success
//
// .../remove/{key}
func (kn *KEONServer) RemoveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		kn.Lock()
		defer kn.Unlock()

		if kn.remove == nil {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			return
		}

		switch r.Method {
		case "GET":

			kn.remove([]byte(filepath.Base(r.URL.Path)))

		case "POST":

			scanner := bufio.NewScanner(r.Body)
			for scanner.Scan() {
				kn.remove(scanner.Bytes())
			}
		}

		w.WriteHeader(http.StatusOK) // 200
	}
}

// check item
//
//	503 unavailable
//	200 success
//
//	.../check/{key}
func (kn *KEONServer) CheckHandler() http.HandlerFunc {

	type Job struct {
		Status bool   `json:"status,omitempty"`
		Key    string `json:"key,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		kn.RLock()
		defer kn.RUnlock()

		if kn.lookup == nil {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			return
		}

		w.WriteHeader(http.StatusOK) // 200
		switch r.Method {
		case "GET":
			var j Job
			j.Key = filepath.Base(r.URL.Path)
			j.Status = kn.lookup([]byte(j.Key))
			json.NewEncoder(w).Encode(j)

		case "POST":
			var j []Job
			scanner := bufio.NewScanner(r.Body)
			for scanner.Scan() {
				j = append(j, Job{
					Key:    scanner.Text(),
					Status: kn.lookup(scanner.Bytes()),
				})
			}
			json.NewEncoder(w).Encode(j)

		}

	}
}
