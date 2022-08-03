package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/makiuchi-d/whitenote/wspace"
	"github.com/pebbe/zmq4"
)

const (
	delimiter = "<IDS|MSG>"
	protoVer  = "5.3"
)

var (
	sessionId  string
	kernelInfo []byte

	metadata  = []byte("{}")
	stateIdle = []byte(`{"execution_state":"idle"}`)
	stateBusy = []byte(`{"execution_state":"busy"}`)
)

func init() {
	sid, _ := uuid.NewRandom()
	sessionId = sid.String()

	kernelInfo, _ = json.Marshal(map[string]any{
		"status":                 "ok",
		"protocol_version":       protoVer,
		"implementation":         "whitenote",
		"implementation_version": "0.1",
		"language_info": map[string]any{
			"name":               "whitespace",
			"version":            "0.1",
			"mimetype":           "", //text/x-whitespace",
			"file_extension":     ".ws",
			"pygments_lexer":     "",
			"codemirror_mode":    "",
			"nbconvert_exporter": "",
		},
		"banner": "",
	})
}

type Sockets struct {
	conf    *ConnectionInfo
	shell   *zmq4.Socket
	control *zmq4.Socket
	stdin   *zmq4.Socket
	iopub   *zmq4.Socket
	hb      *zmq4.Socket
}

type ConnectionInfo struct {
	SignatureScheme string `json:"signature_scheme"`
	Transport       string `json:"transport"`
	StdinPort       int    `json:"stdin_port"`
	ControlPort     int    `json:"control_port"`
	IOPubPort       int    `json:"iopub_port"`
	HBPort          int    `json:"hb_port"`
	ShellPort       int    `json:"shell_port"`
	Key             string `json:"key"`
	IP              string `json:"ip"`
}

type Message struct {
	ZmqID    [][]byte
	Header   []byte
	Parent   []byte
	Metadata []byte
	Content  []byte
	Extra    [][]byte
}

func readConf(file string) *ConnectionInfo {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	var conf ConnectionInfo
	if err := json.Unmarshal(c, &conf); err != nil {
		panic(err)
	}
	return &conf
}

func bindSocket(typ zmq4.Type, transport, ip string, port int) *zmq4.Socket {
	sock, err := zmq4.NewSocket(typ)
	if err != nil {
		panic(err)
	}
	sock.Bind(fmt.Sprintf("%s://%s:%d", transport, ip, port))
	return sock
}

func newSockets(conf *ConnectionInfo) *Sockets {
	return &Sockets{
		conf:    conf,
		shell:   bindSocket(zmq4.ROUTER, conf.Transport, conf.IP, conf.ShellPort),
		control: bindSocket(zmq4.ROUTER, conf.Transport, conf.IP, conf.ControlPort),
		stdin:   bindSocket(zmq4.ROUTER, conf.Transport, conf.IP, conf.StdinPort),
		iopub:   bindSocket(zmq4.PUB, conf.Transport, conf.IP, conf.IOPubPort),
		hb:      bindSocket(zmq4.REP, conf.Transport, conf.IP, conf.HBPort),
	}
}

func calcHMAC(key string, header, parent, metadata, content []byte) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(header)
	h.Write(parent)
	h.Write(metadata)
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Sockets) recvRouterMessage(sock *zmq4.Socket) (*Message, error) {
	mb, err := sock.RecvMessageBytes(0)
	if err != nil {
		return nil, err
	}

	var d int
	for d = 0; d < len(mb); d++ {
		if bytes.Equal(mb[d], []byte(delimiter)) {
			break
		}
	}
	if d > len(mb)-5 {
		return nil, fmt.Errorf("invalid message: %v,%v, %v", d, len(mb), mb)
	}

	msg := &Message{
		ZmqID:    mb[:d],
		Header:   mb[d+2],
		Parent:   mb[d+3],
		Metadata: mb[d+4],
		Content:  mb[d+5],
		Extra:    mb[d+6:],
	}

	sig := string(mb[d+1])
	mac := calcHMAC(s.conf.Key, msg.Header, msg.Parent, msg.Metadata, msg.Content)
	if sig != mac {
		return msg, fmt.Errorf("invalid hmac: %v %v", sig, mb)
	}

	return msg, nil
}

func newHeader(msgtype string) []byte {
	mid, _ := uuid.NewRandom()
	h := map[string]any{
		"date":     time.Now().Format(time.RFC3339),
		"msg_id":   mid.String(),
		"username": "kernel",
		"session":  sessionId,
		"msg_type": msgtype,
		"version":  protoVer,
	}
	hdr, _ := json.Marshal(h)
	return hdr
}

func (s *Sockets) sendState(parent *Message, state []byte) {
	hdr := newHeader("status")
	phdr := parent.Header
	mac := calcHMAC(s.conf.Key, hdr, phdr, metadata, state)
	_, _ = s.iopub.SendMessage(
		delimiter,
		mac,
		hdr,
		phdr,
		metadata,
		state)
}

func (s *Sockets) sendStdout(parent *Message, output string) {
	content, _ := json.Marshal(map[string]string{
		"name": "stdout",
		"text": output,
	})
	hdr := newHeader("stream")
	phdr := parent.Header
	mac := calcHMAC(s.conf.Key, hdr, phdr, metadata, content)
	_, _ = s.iopub.SendMessage(
		delimiter,
		mac,
		hdr,
		phdr,
		metadata,
		content)
}

func (s *Sockets) sendRouter(sock *zmq4.Socket, msgtype string, parent *Message, content []byte) {
	hdr := newHeader(msgtype)
	phdr := parent.Header
	mac := calcHMAC(s.conf.Key, hdr, phdr, metadata, content)
	data := make([]any, 0, len(parent.ZmqID)+6)
	for _, p := range parent.ZmqID {
		data = append(data, p)
	}
	data = append(data, delimiter)
	data = append(data, mac)
	data = append(data, hdr)
	data = append(data, phdr)
	data = append(data, metadata)
	data = append(data, content)
	_, _ = sock.SendMessage(data...)
}

func (s *Sockets) sendExecuteReply(sock *zmq4.Socket, parent *Message, count int) {
	content := fmt.Sprintf(`{"status":"ok","execution_count":%d}`, count)
	s.sendRouter(sock, "execute_reply", parent, []byte(content))
}

func (s *Sockets) shellHandler(vm *wspace.VM) {
	execCount := 0
	for {
		msg, err := s.recvRouterMessage(s.shell)
		if err != nil {
			log.Printf("shell: recv: %v", err)
			continue
		}
		var hdr map[string]any
		if err := json.Unmarshal(msg.Header, &hdr); err != nil {
			log.Printf("shell: header: %v", err)
			continue
		}

		log.Println("shell:", hdr["msg_type"], string(msg.Content))
		switch hdr["msg_type"] {

		case "kernel_info_request":
			s.sendRouter(s.shell, "kernel_info_reply", msg, kernelInfo)
			s.sendState(msg, stateIdle)

		case "execute_request":
			execCount++
			s.sendState(msg, stateBusy)
			s.sendStdout(msg, "whitenote!!")
			s.sendExecuteReply(s.shell, msg, execCount)
			s.sendState(msg, stateIdle)
		}
	}
}

func (s *Sockets) controlHandler(shutdown chan<- struct{}) {
	for {
		msg, err := s.recvRouterMessage(s.control)
		if err != nil {
			log.Printf("control: recv: %v", err)
			continue
		}
		var hdr map[string]any
		if err := json.Unmarshal(msg.Header, &hdr); err != nil {
			log.Printf("control: header: %v", err)
			continue
		}

		log.Println("control:", hdr["msg_type"], string(msg.Content))
		switch hdr["msg_type"] {
		case "shutdown_request":
			shutdown <- struct{}{}
		}
	}
}

func (s *Sockets) hbHandler() {
	for {
		msg, err := s.hb.Recv(0)
		if err == nil {
			_, err = s.hb.Send(msg, 0)
		}
		log.Printf("heartbeat: %v (%v)", msg, err)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Println("need connectioninfo file")
		return
	}
	conf := readConf(os.Args[1])
	socks := newSockets(conf)

	vm := wspace.New()
	shutdown := make(chan struct{}, 1)

	go socks.shellHandler(vm)
	go socks.controlHandler(shutdown)
	go socks.hbHandler()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sig:
	case <-shutdown:
	}
	log.Println("whitenote shuting down...")
}
