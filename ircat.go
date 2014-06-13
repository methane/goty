package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/methane/ircat/goty"
)

func main() {
	var server, nick, channel string

	flag.StringVar(&server, "server", "irc.freenode.org:6667",
		"Server to connect to in format 'irc.freenode.org:6667'")
	flag.StringVar(&nick, "nick", "goty-bot", "IRC nick")
	flag.StringVar(&channel, "chan", "", "IRC channel (without #)")
	flag.BoolVar(&goty.Debug, "debug", false, "debug mode")
	flag.Parse()

	if channel == "" {
		fmt.Fprintln(os.Stderr, "goty: No channnel specified")
		return
	}
	con, err := goty.Dial(server, nick)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goty: %v\n", err)
		return
	}
	in := bufio.NewReader(os.Stdin)

	connClosed := make(chan interface{})

	// Read from IRC and write to stdout.
	go func() {
		for {
			str, ok := <-con.Read
			if !ok {
				connClosed <- nil
				break
			}
			if strings.HasPrefix(str, "PRIVMSG ") || strings.HasPrefix(str, "NOTICE ") {
				str = str[strings.IndexRune(str, ' ')+1:]
				pos := strings.IndexRune(str, ' ')
				if pos > 0 {
					str = str[pos+1:]
					if str[0] == ':' {
						str = str[1:]
					}
					fmt.Println(str)
				}
			}
		}
	}()

	con.Write <- "JOIN #" + channel

	// Read lines from stdin and send to IRC.
	go func() {
		for {
			input, err := in.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "goty: %v\n", err)
				close(con.Write)
				break
			}
			con.Write <- fmt.Sprintf("NOTICE #%s :%s", channel, strings.TrimRight(input, "\r\n "))
		}
	}()

	<-connClosed
	con.Wait()
}
