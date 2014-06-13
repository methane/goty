package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/methane/ircat/goty"
)

var server *string = flag.String("server", "irc.freenode.org:6667",
	"Server to connect to in format 'irc.freenode.org:6667'")
var nick *string = flag.String("nick", "goty-bot", "IRC nick to use")
var chan_ *string = flag.String("chan", "", "Channel to send")

func main() {
	flag.Parse()

	if chan_ == nil || *chan_ == "" {
		fmt.Fprintln(os.Stderr, "goty: No channnel specified")
		return
	}
	var err error
	var con *goty.IRCConn
	if con, err = goty.Dial(*server, *nick); err != nil {
		fmt.Fprintf(os.Stderr, "goty: %v\n", err)
		return
	}
	in := bufio.NewReader(os.Stdin)

	connClosed := make(chan interface{})
	stdinRead := make(chan string)

	go func() {
		for {
			str, ok := <-con.Read
			if !ok {
				connClosed <- nil
				break
			}
			fmt.Printf("<- %s\n", str)
		}
	}()

	con.Write <- "JOIN #" + *chan_

	// Read lines from string
	go func() {
		for {
			input, err := in.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "goty: %v\n", err)
				close(stdinRead)
				break
			}
			stdinRead <- strings.TrimRight(input, "\r\n ")
		}
	}()

main:
	for {
		select {
		case input, ok := <-stdinRead:
			if !ok {
				close(con.Write)
				stdinRead = nil
				continue
			}
			com := fmt.Sprintf("NOTICE #%s %s", *chan_, input)
			fmt.Printf("-> %s\n", com)
			con.Write <- com
		case <-connClosed:
			break main
		}
	}
}
