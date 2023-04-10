package rdisplay

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/kbinani/screenshot"
)

// XVideoProvider implements the rdisplay.Service interface for XServer
type XVideoProvider struct{}

// XScreenGrabber captures video from a X server
type XScreenGrabber struct {
	id         string
	fps        int
	screen     Screen
	frames     chan *image.RGBA
	stop       chan struct{}
	stopStatus bool
}

// CreateScreenGrabber Creates an screen capturer for the X server
func (*XVideoProvider) CreateScreenGrabber(screen Screen, fps int) (ScreenGrabber, error) {
	return &XScreenGrabber{
		id:         uuid.New().String(),
		screen:     screen,
		fps:        fps,
		frames:     make(chan *image.RGBA),
		stop:       make(chan struct{}),
		stopStatus: true, //default true ,can be close
	}, nil
}

// Screens Returns the available screens to capture
func (x *XVideoProvider) Screens() ([]Screen, error) {
	numScreens := screenshot.NumActiveDisplays()
	screens := make([]Screen, numScreens)
	for i := 0; i < numScreens; i++ {
		screens[i] = Screen{
			Index:  i,
			Bounds: screenshot.GetDisplayBounds(i),
		}
	}
	return screens, nil
}

// Frames returns a channel that will receive an image stream
func (g *XScreenGrabber) Frames() <-chan *image.RGBA {
	return g.frames
}

// Start initiates the screen capture loop
func (g *XScreenGrabber) Start() {
	delta := time.Duration(1000/g.fps) * time.Millisecond
	var lastImg *image.RGBA
	files := FileWalk("/data/local/tmp/h264img")
	i := 0
	useMinicap := true
	go func() {
		for {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("[Recovery] panic recovered: %v\n%s]", r, string([]byte{27, 91, 48, 109}))
				}
			}()

			startedAt := time.Now()
			select {
			case <-g.stop:
				close(g.frames)
				return
			default:
				var file string
				if useMinicap {
					files = FileWalk("/data/local/tmp/h264mini")
				}
				if i == 200 {
					i = 0
				}
				if len(files) >= 2 {
					if useMinicap {
						file = files[len(files)-2]
					} else {
						file = files[i]
					}
					//ToDo compare lastImage currentImge md5 equal or not equal,if euqal not send
					img, err := getImage(file)
					ts := ToString(time.Now().UnixNano() / int64(time.Millisecond))
					fmt.Println(i, ts, file)
					if err == nil {
						lastImg = img
					} else {
						img = lastImg
					}
					if img != nil {
						g.frames <- img
					}
				}
				i++
				ellapsed := time.Now().Sub(startedAt)
				sleepDuration := delta - ellapsed
				if sleepDuration > 0 {
					time.Sleep(sleepDuration)
				}
			}
		}
	}()
}

// Stop sends a stop signal to the capture loop
func (g *XScreenGrabber) Stop() {
	if g.stopStatus {
		close(g.stop)
		g.stopStatus = false
	}
}

// Screen returns a pointer to the screen we're capturing
func (g *XScreenGrabber) Screen() *Screen {
	return &g.screen
}

// Fps returns the frames per sec. we're capturing
func (g *XScreenGrabber) Fps() int {
	return g.fps
}

// NewVideoProvider returns an X Server-based video provider
func NewVideoProvider() (Service, error) {
	return &XVideoProvider{}, nil
}

func getImage(filePath string) (*image.RGBA, error) {
	// Decoding JPEG into image.Image
	imgFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer imgFile.Close()
	imgFile.Seek(0, 0)

	//jpg 图片的开始是ffd8,结束是ffd9 //https://github.com/corkami/formats/blob/master/image/jpeg.md
	//xxd examples.jpeg |egrep "ffd9|ff d9"

	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if len(contents) < 4 {
		return nil, errors.New("below 4 byte")
	}

	// Maybe wrong End-Of-Image.
	if !(contents[0] == '\xff' || contents[1] == '\xd8') {
		return nil, err
	}
	if !(contents[len(contents)-1] == '\xd9' && contents[len(contents)-2] == '\xff') {
		return nil, err
	}

	// Decode the JPEG data. If reading from file, create a reader with
	//
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, err
	}

	rgba := imageToRGBA(img)

	return rgba, err
}

func imageToRGBA(src image.Image) *image.RGBA {

	// No conversion needed if image is an *image.RGBA.
	if dst, ok := src.(*image.RGBA); ok {
		return dst
	}

	// Use the image/draw package to convert to *image.RGBA.
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

func superSampling(inputImage image.Image) image.Image {
	rect := inputImage.Bounds()
	width := rect.Size().X
	height := rect.Size().Y
	rect2 := image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X-1, rect.Max.Y-1)
	rgba := image.NewRGBA(rect2)

	for x := 0; x < width-1; x++ {
		for y := 0; y < height-1; y++ {
			var col color.RGBA
			// 座標(x,y)のR, G, B, α の値を取得
			r00, g00, b00, a00 := inputImage.At(x, y).RGBA()
			r01, g01, b01, a01 := inputImage.At(x, y+1).RGBA()
			r10, g10, b10, a10 := inputImage.At(x+1, y).RGBA()
			r11, g11, b11, a11 := inputImage.At(x+1, y+1).RGBA()
			col.R = uint8((uint(uint8(r00)) + uint(uint8(r01)) + uint(uint8(r10)) + uint(uint8(r11))) / 4)
			col.G = uint8((uint(uint8(g00)) + uint(uint8(g01)) + uint(uint8(g10)) + uint(uint8(g11))) / 4)
			col.B = uint8((uint(uint8(b00)) + uint(uint8(b01)) + uint(uint8(b10)) + uint(uint8(b11))) / 4)
			col.A = uint8((uint(uint8(a00)) + uint(uint8(a01)) + uint(uint8(a10)) + uint(uint8(a11))) / 4)
			rgba.Set(x, y, col)
		}
	}

	return rgba.SubImage(rect)
}

// FileWalk def
func FileWalk(fileDir string) []string {
	start, err := os.Stat(fileDir)
	if err != nil || !start.IsDir() {
		fmt.Printf("RecoverFromFile fileWalk no is a dir [%v]", err)
		return nil
	}
	var targets []string
	filepath.Walk(fileDir, func(fpath string, fi os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("RecoverFromFile fileWalk err[%v]", err)
			return nil
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		targets = append(targets, fpath)
		return nil
	})
	return targets
}
