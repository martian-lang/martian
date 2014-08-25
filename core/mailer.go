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
	"strings"
	"text/template"
)

type Mailer struct {
	InstanceName string
	username     string
	password     string
	host         string
	port         int
	notifyEmail  string
	debug        bool
}

func NewMailer(instanceName string, username string, password string, notifyEmail string, debug bool) *Mailer {
	self := &Mailer{}
	self.InstanceName = strings.ToLower(instanceName)
	self.username = username
	self.password = password
	self.host = "smtp.gmail.com"
	self.port = 587
	self.notifyEmail = notifyEmail
	self.debug = debug
	return self
}

type SmtpTemplateData struct {
	From    string
	To      string
	Subject string
	Body    string
	Cc      string
}

const emailTemplate = `From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}

{{.Body}}

cc: {{.Cc}}

Stay fresh,
Mario
`

func (self *Mailer) Sendmail(to []string, subject string, body string) error {
	var doc bytes.Buffer

	if self.debug {
		subject = "[DEBUG - IGNORE] " + subject
	}

	recipients := append([]string{self.notifyEmail}, to...)

	context := &SmtpTemplateData{
		fmt.Sprintf("Mario Lopez <%s>", self.username),
		self.notifyEmail,
		subject,
		body,
		strings.Join(recipients, ", "),
	}
	t := template.New("emailTemplate")
	t, _ = t.Parse(emailTemplate)
	_ = t.Execute(&doc, context)

	auth := smtp.PlainAuth("", self.username, self.password, self.host)
	return smtp.SendMail(
		fmt.Sprintf("%s:%d", self.host, self.port),
		auth,
		self.username,
		recipients,
		doc.Bytes(),
	)
}
