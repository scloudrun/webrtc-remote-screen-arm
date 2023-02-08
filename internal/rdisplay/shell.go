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

func InitCrontab() {
	go remove()
	delta := time.Duration(1000/10) * time.Millisecond
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
			run()
			fmt.Println("here run")
			ellapsed := time.Now().Sub(startedAt)
			sleepDuration := delta - ellapsed
			if sleepDuration > 0 {
				time.Sleep(sleepDuration)
			}
		}
	}
}

func remove() {
	ticker := time.NewTicker(time.Duration(5) * time.Second)
	for {
		fmt.Println("here remove")
		files := FileWalk("./h264mini")
		if len(files) >3 {
			for k,v := range files {
				if k >= len(files) -2 {
					continue
				}
				fmt.Println(v)
				err := RemoveFile(v)
				if err !=nil {
					fmt.Println(err)
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


func run() {
	shellCmd := "LD_LIBRARY_PATH=/data/local/tmp /data/local/tmp/minicap -P \"1080x1920@1080x1920/0\" -s > /data/local/tmp/h264mini/"+ToString(time.Now().UnixNano()/int64(time.Millisecond))+".jpg"
	_,err := RunCommand(shellCmd)
	if err !=nil {
		fmt.Println(time.Now().Unix())
		fmt.Println(err)
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
