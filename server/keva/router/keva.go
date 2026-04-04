package router

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/zxdev/kvs"
)

type KEVAServer struct {
	keva   *kvs.KEVA
	insert func([]byte, uint64) struct{ Ok, Exist, NoSpace bool }
	lookup func(key []byte) (item struct {
		Value uint64
		Ok    bool
	})
	remove func([]byte) struct{ Ok, Exist bool }
	sync.RWMutex
}

// create handler deafults to 1mm object container default
//
//	.../create/{size [1m 10m 100m {n} drop|reset]}
func (kn *KEVAServer) CreateHandler() http.HandlerFunc {

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
			resp.N = a2i(v)
		}

		if resp.N == 0 {
			kn.Lock()
			kn.keva = nil
			kn.insert = nil
			kn.remove = nil
			kn.Unlock()
			resp.Message = "drop"

		} else {
			kn.Lock()
			kn.keva = kvs.NewKEVA(int(resp.N), nil)
			kn.insert = kn.keva.Insert(true)
			kn.lookup = kn.keva.Lookup()
			kn.remove = kn.keva.Remove()
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
//	424 no keva container
//	200 keva statistics
//
// .../status
func (kn *KEVAServer) StatusHandler() http.HandlerFunc {

	type Stats struct {
		Count   int `json:"count,omitempty"`
		Max     int `json:"max,omitempty"`
		Percent int `json:"percent,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		if kn.keva == nil {
			w.WriteHeader(http.StatusFailedDependency) // 424

		} else {

			w.WriteHeader(http.StatusOK) // 200
			json.NewEncoder(w).Encode(&Stats{
				Count:   int(kn.keva.Len()),
				Max:     int(kn.keva.Cap()),
				Percent: int(kn.keva.Ratio()),
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
//	.../insert/{ string:uint64 }
func (kn *KEVAServer) InsertHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		kn.Lock()
		defer kn.Unlock()

		if kn.insert == nil {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			return
		}

		switch r.Method {
		case "GET":
			var v uint64
			var dat = strings.Split(filepath.Base(r.URL.Path), ":")
			if len(dat) == 2 {
				v = a2i(dat[1])
			}
			if !kn.insert([]byte(strings.TrimSpace(dat[0])), v).Ok {
				w.WriteHeader(http.StatusBadRequest) // 400
				return
			}

		case "POST":

			var v uint64
			var dat []string
			scanner := bufio.NewScanner(r.Body)
			for scanner.Scan() {
				v = 0
				dat = strings.Split(filepath.Base(r.URL.Path), ":")
				if len(dat) == 2 {
					v = a2i(dat[1])
				}
				if !kn.insert([]byte(dat[0]), v).Ok {
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
//	.../remove/{key}
func (kn *KEVAServer) RemoveHandler() http.HandlerFunc {
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
func (kn *KEVAServer) CheckHandler() http.HandlerFunc {

	type Job struct {
		Status bool   `json:"status,omitempty"`
		Key    string `json:"key,omitempty"`
		Value  uint64 `json:"value,omitempty"`
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
			result := kn.lookup([]byte(j.Key))
			j.Status = result.Ok
			j.Value = result.Value
			json.NewEncoder(w).Encode(&j)

		case "POST":
			var j []Job
			scanner := bufio.NewScanner(r.Body)
			for scanner.Scan() {
				result := kn.lookup(scanner.Bytes())
				j = append(j, Job{
					Status: result.Ok,
					Key:    scanner.Text(),
					Value:  result.Value,
				})
			}
			json.NewEncoder(w).Encode(&j)

		}

	}
}
