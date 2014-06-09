package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/methane/goty"
)

var server *string = flag.String("server", "irc.freenode.org:6667", "Server to connect to in format 'irc.freenode.org:6667'")
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

	go func() {
		for {
			str, ok := <-con.Read
			if !ok {
				break
			}
			fmt.Printf("<- %s\n", str)
		}
	}()

	con.Write <- "JOIN " + *chan_

	for {
		input, err := in.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "goty: %v\n", err)
			break
		}
		fmt.Printf("-> %s", input)
		con.Write <- "NOTICE " + *chan_ + input[:len(input)-1]
	}
	if err := con.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "goty: %v\n", err)
	}
}
