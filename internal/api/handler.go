package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/scloudrun/webrtc-remote-screen-arm/internal/rdisplay"
	"github.com/scloudrun/webrtc-remote-screen-arm/internal/rtc"
)

var webrtcConnMap = map[string]interface{}{}

func handleError(w http.ResponseWriter, err error) {
	fmt.Printf("Error: %v", err)
	w.WriteHeader(http.StatusInternalServerError)
}

// MakeHandler returns an HTTP handler for the session service
func MakeHandler(webrtc rtc.Service, display rdisplay.Service, frameCount int) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		dec := json.NewDecoder(r.Body)
		req := newSessionRequest{}

		if err := dec.Decode(&req); err != nil {
			handleError(w, err)
			return
		}

		peer, err := webrtc.CreateRemoteScreenConnection(req.Screen, frameCount)
		if err != nil {
			handleError(w, err)
			return
		}

		answer, err := peer.ProcessOffer(req.Offer)

		if err != nil {
			handleError(w, err)
			return
		}

		payload, err := json.Marshal(newSessionResponse{
			Answer: answer,
		})
		if err != nil {
			handleError(w, err)
			return
		}

		ip := "127.0.0.1"
		if webrtcConnMap[ip] != nil {
			v := webrtcConnMap[ip].(rtc.RemoteScreenConnection)
			v.Close()
		}
		rdisplay.RunStatus = true
		webrtcConnMap[ip] = peer
		w.Write(payload)
	})

	mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ip := "127.0.0.1"
		if webrtcConnMap[ip] == nil {
			return
		}
		v := webrtcConnMap[ip].(rtc.RemoteScreenConnection)
		v.Close()
		rdisplay.RunStatus = false
		webrtcConnMap[ip] = nil
	})

	mux.HandleFunc("/screens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		/**
				screens, err := display.Screens()

				if err != nil {
					handleError(w, err)
					return
				}

				screensPayload := make([]screenPayload, len(screens))

				for i, s := range screens {
					screensPayload[i].Index = s.Index
				}
				payload, err := json.Marshal(screensResponse{
					Screens: screensPayload,
				})
				if err != nil {
					handleError(w, err)
					return
				}
		**/
		payload := []byte("{\"screens\":[{\"index\":0}]}")
		w.Write(payload)
	})
	return mux
}
