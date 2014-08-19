//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// Mario mailer.
//
package core

import (
	"bytes"
	"fmt"
	"net/smtp"
	"text/template"
)

type Mailer struct {
	username string
	password string
	host     string
	port     int
}

func NewMailer(username string, password string) *Mailer {
	self := &Mailer{}
	self.username = username
	self.password = password
	self.host = "smtp.gmail.com"
	self.port = 587
	return self
}

type SmtpTemplateData struct {
	From    string
	To      string
	Subject string
	Body    string
}

const emailTemplate = `From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}

{{.Body}}

Kapow,
M.Lo
`

func (self *Mailer) Sendmail(subject string, body string) error {
	var doc bytes.Buffer
	to := "alex@10xtechnologies.com"

	context := &SmtpTemplateData{
		fmt.Sprintf("Mario Lopez <%s>", self.username),
		to,
		subject,
		body,
	}
	t := template.New("emailTemplate")
	t, _ = t.Parse(emailTemplate)
	_ = t.Execute(&doc, context)

	auth := smtp.PlainAuth("", self.username, self.password, self.host)
	return smtp.SendMail(
		fmt.Sprintf("%s:%d", self.host, self.port),
		auth,
		self.username,
		[]string{to},
		doc.Bytes(),
	)
}
