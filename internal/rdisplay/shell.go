package rdisplay

import (
	"bytes"
	"fmt"
	"os/exec"
	"os"
	"time"
	"strconv"

	"github.com/bitfield/script"
)

// RunShell def
func RunShell(cmd string) (string, error) {
	p := script.Exec(cmd)
	output, err := p.String()
	p.Close()
	return output, err
}

//ShellToUse def
const ShellToUse = "sh"
const h264Path = "./h264mini"
var RunStatus = false

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
		fmt.Printf("[err] : %v\n]",err)
	}
	return stdout.String(), err
}

func InitCrontab(frameCount int) {
	RemoveFile(h264Path)
	CreateDir(h264Path)
	go remove()
	delta := time.Duration(1000/frameCount) * time.Millisecond
	signals := make(chan bool)
	for {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[Recovery] panic recovered: %v\n%s]", r, string([]byte{27, 91, 48, 109}))
			}
		}()

		startedAt := time.Now()
		select {
		case <-signals:
			return
		default:
			if RunStatus {
				run()
			}
			ellapsed := time.Now().Sub(startedAt)
			sleepDuration := delta - ellapsed
			if sleepDuration > 0 {
				time.Sleep(sleepDuration)
			}
		}
	}
}

func remove() {
	ticker := time.NewTicker(time.Duration(2) * time.Second)
	for {
		files := FileWalk(h264Path)
		if len(files) >3 {
			for k,v := range files {
				if k >= len(files) -2 {
					continue
				}
				err := RemoveFile(v)
				if err !=nil {
					fmt.Printf("[err] : %v\n]",err)
				}
			}
		}
		<-ticker.C
	}
}


//RemoveFile def
func RemoveFile(path string) error {
	err := os.RemoveAll(path)
	return err
}

//CreateDir def
func CreateDir(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
		os.Chmod(path, 0755)
	}
	return true
}

/**
minicap 内部流程解析
(1) 解析 LD_LIBRARY_PATH=/data/local/tmp /data/local/tmp/minicap -P 1440x2560@1440x2560/0 启动行参数，设置相关配置。

  其中：
   -P：(x@x/{0|90|180|270}) minicap 产生的图片尺寸以及传递出来的图片尺寸,可修改该参数调整 client 收到的图片的宽度或高度。eg：1080x1920@360x640/0 指定 minicap 在手机内部产生图片分辨率为 1080x1920，而传递出来的图片分辨率为 360x640,0 表示手机目前是竖屏状态，逆时针旋转手机可根据不同角度设置此参数为 90、180、270。
   -Q： 设置 minicap 内部对图片的压缩比：0—100
  注：minicap 内部会对期望图片的分辨率做一个调整，如果期望的分辨率大小和手机的真实分辨率大小是等比的就不做调整，如果不等比则会调整期望的图片的分辨率大小。
**/
func run() {
	shellCmd := "LD_LIBRARY_PATH=/data/local/tmp /data/local/tmp/minicap -Q 70 -P \"1080x1920@540x960/0\" -s > /data/local/tmp/h264mini/"+ToString(time.Now().UnixNano()/int64(time.Millisecond))+".jpg"
	_,err := RunCommand(shellCmd)
	if err !=nil {
		fmt.Printf("[err] : %v\n]",err)
	}
}

// ToString def
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	switch v.(type) {
	case int:
		return strconv.FormatInt(int64(v.(int)), 10)
	case uint:
		return strconv.FormatInt(int64(v.(uint)), 10)
	case int64:
		return strconv.FormatInt(v.(int64), 10)
	}

	return ""
}
