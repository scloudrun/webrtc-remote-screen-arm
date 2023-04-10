package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/scloudrun/webrtc-remote-screen-arm/internal/api"
	"github.com/scloudrun/webrtc-remote-screen-arm/internal/encoders"
	"github.com/scloudrun/webrtc-remote-screen-arm/internal/rdisplay"
	"github.com/scloudrun/webrtc-remote-screen-arm/internal/rtc"
)

const (
	httpDefaultPort = "8020"
	//defaultStunServer = "stun:stun.l.google.com:19302"
	defaultStunServer = "stun:172.24.206.96:8021"
	defaultFrameCount = 5
	openStatus        = 0
)

func main() {

	httpPort := flag.String("http.port", httpDefaultPort, "HTTP listen port")
	stunServer := flag.String("stun.server", defaultStunServer, "STUN server URL (stun:)")
	frameCount := flag.Int("frame.count", defaultFrameCount, "frame count like 10 ,5")
	openMinicapStatus := flag.Int("minicap.status", openStatus, "1 open ,o not open")
	flag.Parse()

	var video rdisplay.Service
	video, err := rdisplay.NewVideoProvider()
	if err != nil {
		log.Fatalf("Can't init video: %v", err)
	}
	_, err = video.Screens()
	if err != nil {
		log.Fatalf("Can't get screens: %v", err)
	}

	var enc encoders.Service = &encoders.EncoderService{}
	if err != nil {
		log.Fatalf("Can't create encoder service: %v", err)
	}

	if *openMinicapStatus == 1 {
		rdisplay.OpenStatus = true
	}

	var webrtc rtc.Service
	webrtc = rtc.NewRemoteScreenService(*stunServer, video, enc)

	mux := http.NewServeMux()

	// Endpoint to create a new speech to text session
	mux.Handle("/api/", http.StripPrefix("/api", api.MakeHandler(webrtc, video, *frameCount)))

	// Serve static assets
	mux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./web"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, "./web/index.html")
	})
	go rdisplay.InitCrontab(*frameCount)
	errors := make(chan error, 2)
	go func() {
		log.Printf("Starting signaling server on port %s", *httpPort)
		errors <- http.ListenAndServe(fmt.Sprintf(":%s", *httpPort), mux)
	}()

	go func() {
		interrupt := make(chan os.Signal)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
		errors <- fmt.Errorf("Received %v signal", <-interrupt)
	}()

	err = <-errors
	log.Printf("%s, exiting.", err)
}
