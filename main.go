/*
 * Parts are taken from Andy Balholm's grayland (https://github.com/andybalholm/grayland).
 * For those parts the follwoing copyright applies:
 *   Copyright (c) 2016 Andy Balholm. All rights reserved.
 */
package main

import (
	"bytes"
	"flag"
	"net"
	"net/textproto"
	"os"

	"github.com/steffentemplin/emailauth"

	"github.com/andybalholm/milter"
)

var (
	listenAddr = flag.String("listen", "", "address to listen on (instead of using inetd)")
)

func main() {
	flag.Parse()

	var listener net.Listener
	var err error

	if *listenAddr == "" {
		// inetd mode
		listener, err = net.FileListener(os.Stdin)
		if err != nil {
			Fatal("Could not get listener socked from inetd (run this program from inetd or use the --listen option)", "error", err)
		}
	} else {
		// listening mode
		listener, err = net.Listen("tcp", *listenAddr)
		if err != nil {
			Fatal("Could not open listening socket", "address", *listenAddr, "error", err)
		}
	}

	err = milter.Serve(listener, func() milter.Milter { return newAuthMilter() })
	if err != nil {
		Fatal("Error while running milter", "error", err)
	}
}

type authMilter struct {
	ClientHostname string
	ClientIP       net.IP
	HeloName       string
	EnvFrom        string
	EnvTo          []string
	Message        *emailauth.Message

	SPFResult *emailauth.SPFResult
}

func newAuthMilter() *authMilter {
	m := &authMilter{}
	m.EnvTo = make([]string, 1)
	m.Message = new(emailauth.Message)
	return m
}

func (m *authMilter) Connect(hostname string, network string, address string, macros map[string]string) milter.Response {
	switch network {
	case "tcp4", "tcp6":
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			// TODO: really?
			Log("Missing port number in client address", "address", address)
			return milter.Accept
		}
		if host == "127.0.0.1" {
			Log("Connection from localhost")
			return milter.Accept
		}

		m.ClientHostname = hostname
		m.ClientIP = net.ParseIP(host)
	default:
		Log("Non-TCP connection", "network", network, "address", address)
		return milter.Accept
	}

	return milter.Continue
}

func (m *authMilter) Helo(name string, macros map[string]string) milter.Response {
	m.HeloName = name
	return milter.Continue
}

func (m *authMilter) From(sender string, macros map[string]string) milter.Response {
	m.EnvFrom = sender
	spf := new(emailauth.SPFValidator)
	result := spf.Validate(m.ClientIP, m.EnvFrom, m.HeloName)
	m.SPFResult = result
	// TODO: Enforce policy
	return milter.Continue
}

func (m *authMilter) To(recipient string, macros map[string]string) milter.Response {
	m.EnvTo = append(m.EnvTo, recipient)
	// TODO: One header per recipient? How?
	return milter.Continue
}

func (m *authMilter) Headers(h textproto.MIMEHeader) milter.Response {
	m.Message.Headers = &h
	return milter.Continue
}

func (m *authMilter) Body(body []byte, modifier milter.Modifier) milter.Response {
	// TODO: improve via Reader
	m.Message.Body = bytes.NewReader(body)

	dkimValidator := new(emailauth.DKIMValidator)
	dkim := dkimValidator.Validate(m.Message)

	dmarcValidator := new(emailauth.DMARCValidator)
	dmarc := dmarcValidator.Validate(m.Message, m.SPFResult, dkim)

	// TODO: clear headers

	modifier.AddHeader("X-Test-SPF", m.SPFResult.Result.String())
	modifier.AddHeader("X-Test-DKIM", dkim.Result.String())
	modifier.AddHeader("X-Test-DMARC", dmarc.Result.String())

	// TODO: Authentication-Results

	return milter.Continue
}
