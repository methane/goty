package goty

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var Debug = false

var numericReplyExp *regexp.Regexp = regexp.MustCompile(`\A(\d\d\d) .*`)

type IRCConn struct {
	sync.WaitGroup
	Sock        *net.TCPConn
	Read, Write chan string
}

func Dial(server, nick string) (*IRCConn, error) {
	read := make(chan string)
	write := make(chan string)
	con := &IRCConn{
		Sock: nil,
		Read: read, Write: write}
	err := con.Connect(server, nick)
	return con, err
}

func (con *IRCConn) Connect(server, nick string) error {
	var raddr *net.TCPAddr
	var err error
	if raddr, err = net.ResolveTCPAddr("tcp", server); err != nil {
		return err
	}
	if con.Sock, err = net.DialTCP("tcp", nil, raddr); err != nil {
		return err
	}

	r := bufio.NewReader(con.Sock)
	w := bufio.NewWriter(con.Sock)
	nickSuccess := make(chan interface{})
	con.Add(2)

	go func() {
		defer con.Done()
		nickNum := 1
		nickSucceed := false

		for {
			str, err := r.ReadString(byte('\n'))
			if err != nil {
				fmt.Fprintf(os.Stderr, "goty: read: %#v\n", err)
				close(con.Read)
				break
			}
			if Debug {
				fmt.Fprintf(os.Stderr, "<- %#v\n", str)
			}
			s := str
			if strings.HasPrefix(s, ":") {
				if index := strings.IndexRune(s, ' '); index != -1 {
					s = s[index+1:]
				}
			}
			if strings.HasPrefix(s, "PING") {
				con.Write <- "PONG" + str[4:len(str)-2]
				continue
			}
			if m := numericReplyExp.FindStringSubmatch(s); m != nil {
				switch m[1] {
				// 433 - ERR_NICKNAMEINUSE
				// 436 - ERR_NICKCOLLISION
				// 437 - ERR_UNAVAILRESOURCE
				case "433", "436", "437":
					con.Write <- fmt.Sprintf("NICK %s%d", nick, nickNum)
					nickNum++
					continue
				case "001":
					close(nickSuccess)
					nickSucceed = true
					continue
				}
			}
			if nickSucceed {
				con.Read <- s[0 : len(s)-2]
			}
		}
	}()

	go func() {
		defer func() {
			con.Sock.CloseWrite()
			con.Done()
		}()
		for {
			str, ok := <-con.Write
			if !ok {
				if Debug {
					fmt.Fprintf(os.Stderr, "goty: write closed\n")
				}
				break
			}
			if Debug {
				fmt.Fprintf(os.Stderr, "-> %#v\n", str)
			}
			if _, err := w.WriteString(str + "\r\n"); err != nil {
				fmt.Fprintf(os.Stderr, "goty: write: %v\n", err)
				break
			}
			w.Flush()
		}
	}()

	time.Sleep(time.Second)
	con.Write <- "NICK " + nick
	con.Write <- "USER bot * * :..."
	<-nickSuccess
	return nil
}
