package rtc

import (
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"time"

	"github.com/scloudrun/webrtc-remote-screen-arm/internal/encoders"
	"github.com/scloudrun/webrtc-remote-screen-arm/internal/rdisplay"

	"github.com/nfnt/resize"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

var fileNumber = 1

func resizeImage(src *image.RGBA, target image.Point) *image.RGBA {
	return resize.Resize(uint(target.X), uint(target.Y), src, resize.Lanczos3).(*image.RGBA)
}

type rtcStreamer struct {
	track      *webrtc.TrackLocalStaticSample
	stop       chan struct{}
	stopStatus bool
	screen     *rdisplay.ScreenGrabber
	encoder    *encoders.Encoder
	size       image.Point
	count      int
}

func newRTCStreamer(track *webrtc.TrackLocalStaticSample, screen *rdisplay.ScreenGrabber, encoder *encoders.Encoder, size image.Point) videoStreamer {
	return &rtcStreamer{
		track:      track,
		stop:       make(chan struct{}),
		stopStatus: true, //default true ,can be close
		screen:     screen,
		encoder:    encoder,
		size:       size,
		count:      0,
	}
}

func (s *rtcStreamer) initEncoder() error {
	var (
		enc encoders.Service = &encoders.EncoderService{}
	)

	screen := rdisplay.Screen{Index: 0, Bounds: image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{1920, 1080}}}
	sourceSize := image.Point{
		screen.Bounds.Dx(),
		screen.Bounds.Dy(),
	}

	if v, err := enc.NewEncoder(1, sourceSize, 10); err != nil {
		return err
	} else {
		s.encoder = &v
	}
	return nil
}

func (s *rtcStreamer) start() {
	go s.startStream()
}

func (s *rtcStreamer) startStream() {
	screen := *s.screen
	screen.Start()
	frames := screen.Frames()
	for {
		select {
		case <-s.stop:
			time.Sleep(time.Second * 1)
			screen.Stop()
			return
		case frame := <-frames:
			err := s.stream(frame)
			if err != nil {
				fmt.Printf("Streamer: %v\n", err)
				return
			}
		}
	}
}

var fileNumberMap = map[string]int{}

func (s *rtcStreamer) stream(frame *image.RGBA) error {
	resized := resizeImage(frame, s.size)
	if s.count%50 == 0 {
		s.count = 0
		(*s.encoder).Close()
		s.initEncoder()
		if *s.encoder == nil {
			return nil
		}
	}
	payload, err := (*s.encoder).Encode(resized)
	if err != nil {
		return err
	}
	if payload == nil {
		return nil
	}
	s.count++
	return s.track.WriteSample(media.Sample{
		Data:     payload,
		Duration: time.Second,
	})
}

func (s *rtcStreamer) close() {
	if s.stopStatus {
		close(s.stop)
		s.stopStatus = false
	}
}

// Write def
func Write(name string, content []byte) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(content)
	return err
}

// GetFileByte def
func GetFileByte(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil && os.IsNotExist(err) {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}
