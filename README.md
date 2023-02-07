## WebRTC remote view for arm


### Thanks rviscarra/webrtc-remote-screen 


### CN 中文
- 支持arm android手机运行
- 通过对手机上内置好的截图进行h264编码传输
- 支持webrtc v3版本已测试通过

### CN 自行部署
- 调整agent.go中turn地址
- 调整web js中turn地址
- 执行build.sh

- [Go 1.19](https://golang.org/doc/install)
- If you want h264 support: libx264 (included in x264-go, you'll need a C compiler / assembler to build it)
- If you want VP8 support: libvpx


### Links
- https://github.com/pion/webrtc/tree/master/examples/ice-single-port
- https://github.com/aler9/rtsp-simple-server
- termux tool(use android arm build this project)

### Running the server

The server receives the following flags through the command line:

`--http.port` (Optional) 

Specifies the port where the HTTP server should listen, by default the port 9000 is used.

`--stun.server` (Optional)

Allows to speficy a different [STUN](https://wikipedia.org/wiki/STUN) server, by default a Google STUN server is used.

Chrome 74+, Firefox 66+, Safari 12.x are supported. Older versions (within reason) should be supported as well but YMMV.

### Building the server

Build the _deployment_ package by runnning `make`. This should create a tar file with the 
binary and web directory, by default only support for h264 is included, if you want to use VP8 run `make encoders=vp8`, if you want both then `make encoders=vp8,h264`.

Copy the archive to a remote server, decompress it and run `./agent`. The `agent` application assumes the web dir. is in the same directory. 

WebRTC requires a _secure_ domain to work, the recommended approach towards this is to forward the agent port thru SSH tunneling:

```bash
ssh -L YOUR_LOCAL_PORT:localhost:9000 
```

Then access the application on `http://localhost:YOUR_LOCAL_PORT`, localhost should be considered 
secure by modern browsers.

