package goty

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
)

var numericReplyExp *regexp.Regexp = regexp.MustCompile(`\A(\d\d\d) .*`)

type IRCConn struct {
	Sock        *net.TCPConn
	Read, Write chan string
}

func Dial(server, nick string) (*IRCConn, error) {
	read := make(chan string, 1000)
	write := make(chan string, 1000)
	con := &IRCConn{nil, read, write}
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

	go func() {
		nickNum := 1
		for {
			var str string
			if str, err = r.ReadString(byte('\n')); err != nil {
				fmt.Fprintf(os.Stderr, "goty: read: %s\n", err.Error())
				break
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
					con.Write <- fmt.Sprintf("%s%d", nick, nickNum)
					nickNum++
					continue
				}
			}
			con.Read <- str[0 : len(str)-2]
		}
	}()

	go func() {
		for {
			str, ok := <-con.Write
			if !ok {
				fmt.Fprintf(os.Stderr, "goty: write closed\n")
				break
			}
			if _, err := w.WriteString(str + "\r\n"); err != nil {
				fmt.Fprintf(os.Stderr, "goty: write: %s\n", err.Error())
				break
			}
			w.Flush()
		}
	}()

	con.Write <- "NICK " + nick
	con.Write <- "USER bot * * :..."
	return nil
}

func (con *IRCConn) Close() error {
	close(con.Read)
	close(con.Write)
	return con.Sock.Close()
}
