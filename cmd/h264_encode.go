package main

import (
	"github.com/scloudrun/webrtc-remote-screen-arm/internal/encoders"
	"github.com/scloudrun/webrtc-remote-screen-arm/internal/rdisplay"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/nfnt/resize"
	"os/exec"

	"github.com/bitfield/script"
)

//go build -tags "h264enc" cmd/h264_encode.go

var (
	Conn    net.Conn
	Encoder encoders.Encoder
)

func main() {
	//initTcp()
	initEncoder()

	size, err := Encoder.VideoSize()
	fmt.Println(size,err)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("here init")
	files := FileWalk("./h264img")
	for i := 0; i < 10; i++ {
		for _, v := range files {
			var err error
			startedAt := time.Now()
			frame, _ := getImage(v) //30ms - 50ms
			ellapsed := time.Now().Sub(startedAt)
			resized := resizeImage(frame, size) //80ms-110ms
			ellapsedResize := time.Now().Sub(startedAt)
			payload, err := Encoder.Encode(resized) //50ms - 100ms
			ellapsedEncode := time.Now().Sub(startedAt)
			fmt.Printf("diff time[%v] getImage[%v] resized[%v] encode[%v] err[%v]\n", time.Now().UnixNano()/int64(time.Millisecond),ellapsed,ellapsedResize,ellapsedEncode,err)
			debug := false
			if debug {
				payload = getHeaderByte(payload)
				Conn.Write(payload)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func initEncoder() {
	var (
		err error
		enc encoders.Service = &encoders.EncoderService{}
	)

	screen := rdisplay.Screen{Index: 0, Bounds: image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{1920, 1080}}}
	sourceSize := image.Point{
		screen.Bounds.Dx(),
		screen.Bounds.Dy(),
	}

	Encoder, err = enc.NewEncoder(1, sourceSize, 10)
	if err != nil {
		panic(err)
		return
	}
}

func initTcp() {
	var err error
	hostInfo := "172.24.206.71:10001"
	Conn, err = net.Dial("tcp", hostInfo)
	if err != nil {
		panic(err)
	}

	if Conn != nil {
		err = Conn.(*net.TCPConn).SetKeepAlive(true)
		err = Conn.(*net.TCPConn).SetKeepAlivePeriod(30 * time.Second)
	}
	fmt.Println("init start client conn ", Conn, err)

}

func resizeImage(src *image.RGBA, target image.Point) *image.RGBA {
	return resize.Resize(uint(target.X), uint(target.Y), src, resize.Lanczos3).(*image.RGBA)
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
	/**
	for _, target := range targets {
		fmt.Printf("RecoverFromFile fileWalk file[%s]", target)
	}
	**/
	return targets
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
	//xxd examples_flow.jpeg |egrep "ffd9|ff d9"

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

func getHeaderByte(body []byte) (totalB []byte) {
	dataLen := len(body)
	for i := 0; i < 8; i++ {
		var b []byte
		if i == 7 {
			b = Int32ToBytes(uint32(dataLen))
		} else if i == 0 {
			b = []byte("V#V#")
		} else if i == 1 {
			b = Int32ToBytes(120)
		} else {
			b = Int32ToBytes(1)
		}
		totalB = bytesMerge(totalB, b)
	}
	totalB = bytesMerge(totalB, body)
	return
}

func getByteByImg(path string, typ int) (totalB []byte) {
	data := readImg(path)
	if data == nil || len(data) == 0 {
		return totalB
	}
	dataLen := len(data)
	for i := 0; i < 8; i++ {
		var b []byte
		if i == 7 {
			b = Int32ToBytes(uint32(dataLen))
		} else if i == 0 {
			b = []byte("V#V#")
		} else if i == 1 {
			b = Int32ToBytes(120)
		} else {
			b = Int32ToBytes(1)
		}
		totalB = bytesMerge(totalB, b)
	}
	totalB = bytesMerge(totalB, data)
	return
}

func readImg(path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("read file fail", err)
		return nil
	}
	defer f.Close()
	fd, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("read to fd fail", err)
		return nil
	}

	return fd
}

// BytesToUInt32 ...
func BytesToUInt32(buf []byte) uint32 {
	return uint32(binary.BigEndian.Uint32(buf))
}

// Bytes2Str ...
func Bytes2Str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Int32ToBytes ...
func Int32ToBytes(i uint32) []byte {
	var buf = make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

func bytesMerge(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}

//整形转换成字节
func IntToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

//字节转换成整形
func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return int(x)
}

type HeaderPacket struct {
	Version  int
	Type     int
	Compress int
	Checksum int
	Reverse1 int
	Reverse2 int
	Reverse3 int
	DataSize int
}


// RunShell def
func RunShell(cmd string) (string, error) {
	p := script.Exec(cmd)
	output, err := p.String()
	p.Close()
	return output, err
}

//ShellToUse def
const ShellToUse = "sh"

//RunCommand def
func RunCommand(command string) (string, error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
	return stdout.String(), err
}

