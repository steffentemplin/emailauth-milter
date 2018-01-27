package main

import (
	"bufio"
	"net/textproto"
	"strings"
	"testing"

	"github.com/andybalholm/milter"
)

const mailHeaders = "Message-ID: <650131.529753224-sendEmail@sender>\r\n" +
	"From: \"Mail Sender\" <sender@example.com>\r\n" +
	"To: \"Mail Recipient\" <recipient@acme.org>\r\n" +
	"Subject: Mail Nr. #23\r\n" +
	"Date: Sat, 27 Jan 2018 09:34:44 +0000\r\n" +
	"X-Mailer: sendEmail-1.56\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/related; boundary=\"----MIME delimiter for sendEmail-276650.382512347\"\r\n" +
	"\r\n"

const mailBody = "This is a multi-part message in MIME format. To properly display this message you need a MIME-Version 1.0 compliant Email program.\r\n" +
	"\r\n" +
	"------MIME delimiter for sendEmail-276650.382512347\r\n" +
	"Content-Type: text/plain;\r\n" +
	"    charset=\"iso-8859-1\"\r\n" +
	"Content-Transfer-Encoding: 7bit\r\n" +
	"\r\n" +
	"This is an email!\r\n" +
	"\r\n" +
	"------MIME delimiter for sendEmail-276650.382512347--\r\n"

func TestSMTPFlow(t *testing.T) {
	macros := make(map[string]string)
	m := newAuthMilter()
	response := m.Connect("", "tcp4", "192.168.23.4:25", macros)
	assertResponse(response, milter.Continue, t)

	response = m.Helo("mailout.example.com", macros)
	assertResponse(response, milter.Continue, t)

	response = m.From("<sender@example.com>", macros)
	assertResponse(response, milter.Continue, t)

	response = m.To("<recipient@acme.org>", macros)
	assertResponse(response, milter.Continue, t)

	headerReader := textproto.NewReader(bufio.NewReader(strings.NewReader(mailHeaders)))
	headers, err := headerReader.ReadMIMEHeader()
	if err != nil {
		t.Errorf("Header parsing failed: %s", err)
	}
	response = m.Headers(headers)
	assertResponse(response, milter.Continue, t)

	modifier := NewTestModifier()
	response = m.Body([]byte(mailBody), modifier)
	assertResponse(response, milter.Continue, t)

	for _, header := range []string{"X-Test-SPF", "X-Test-DKIM", "X-TestDMARC"} {
		if _, ok := modifier.addedHeaders[header]; !ok {
			t.Errorf("Missing header %s", "X-Test-SPF")
		}
	}
}

func assertResponse(actual milter.Response, expected milter.Response, t *testing.T) {
	if actual != expected {
		t.Errorf("Expected continue, got %s", actual)
	}
}

type TestModifier struct {
	addedHeaders map[string]string
}

func NewTestModifier() *TestModifier {
	m := &TestModifier{}
	m.addedHeaders = make(map[string]string)
	return m
}

func (t TestModifier) AddRecipient(r string) {

}

func (t TestModifier) DeleteRecipient(r string) {

}

func (t TestModifier) ReplaceBody(newBody []byte) {

}

func (t TestModifier) AddHeader(name, value string) {
	t.addedHeaders[name] = value
}

func (t TestModifier) ChangeHeader(name string, index int, value string) {

}
