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
	host         string
	port         int
	senderEmail  string
	notifyEmail  string
	debug        bool
}

func NewMailer(instanceName string, smtphost string, senderEmail string, notifyEmail string,
	debug bool) *Mailer {

	self := &Mailer{}
	self.InstanceName = strings.ToLower(instanceName)
	self.host = smtphost
	self.port = 25
	self.senderEmail = senderEmail
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

	// If debug mode, put name of instance in subject line.
	if self.debug {
		subject = fmt.Sprintf("[DEBUG - %s] %s", strings.ToUpper(self.InstanceName), subject)
	}

	// Only add individual recipients if not in debug mode.
	recipients := []string{self.notifyEmail}
	if !self.debug {
		recipients = append(recipients, to...)
	}

	// Build template context.
	context := &SmtpTemplateData{
		fmt.Sprintf("Mario Lopez <%s>", self.senderEmail),
		self.notifyEmail,
		subject,
		body,
		strings.Join(recipients, ", "),
	}

	// Render the template.
	t := template.New("emailTemplate")
	t, _ = t.Parse(emailTemplate)
	_ = t.Execute(&doc, context)

	// Send email.
	return smtp.SendMail(
		fmt.Sprintf("%s:%d", self.host, self.port),
		nil,
		self.senderEmail,
		recipients,
		doc.Bytes(),
	)
}
