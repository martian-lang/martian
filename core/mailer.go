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
	InstanceName string
	username     string
	password     string
	host         string
	port         int
	notifyEmail  string
}

func NewMailer(instanceName string, username string, password string, notifyEmail string) *Mailer {
	self := &Mailer{}
	self.InstanceName = instanceName
	self.username = username
	self.password = password
	self.host = "smtp.gmail.com"
	self.port = 587
	self.notifyEmail = notifyEmail
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

Stay fresh,
Mario
`

func (self *Mailer) Sendmail(subject string, body string) error {
	var doc bytes.Buffer

	context := &SmtpTemplateData{
		fmt.Sprintf("Mario Lopez <%s>", self.username),
		self.notifyEmail,
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
		[]string{self.notifyEmail},
		doc.Bytes(),
	)
}
