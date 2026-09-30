package util

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/nexsoft-git/nexlogger/log"
)

type EmailUtility struct {
	host     string
	port     int
	email    string
	password string
	alias    string
	timeout  time.Duration
}

func (e *EmailUtility) Alias(alias string) *EmailUtility {
	e.alias = alias
	return e
}

func (e *EmailUtility) Timeout(duration time.Duration) *EmailUtility {
	e.timeout = duration
	return e
}

func (e *EmailUtility) CompleteEmail(
	param interface{},
	message string,
) string {

	if param == nil || message == "" {
		return message
	}

	v := reflect.ValueOf(param)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return message
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return message
	}

	return replaceFromStructTags(v, message)

}

func replaceFromStructTags(v reflect.Value, message string) string {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		tag := sf.Tag.Get("field")
		fv := v.Field(i)

		if fv.Kind() == reflect.Ptr && !fv.IsNil() {
			elem := fv.Elem()
			if elem.IsValid() && elem.Kind() == reflect.Struct && elem.Type().PkgPath() != "time" {
				message = replaceFromStructTags(elem, message)
			}
		} else if fv.Kind() == reflect.Struct && fv.Type().PkgPath() != "time" {
			message = replaceFromStructTags(fv, message)
		}

		if tag == "" {
			continue
		}

		valStr := valueToString(fv)

		pat := fmt.Sprintf(`\{\{\s*\.%s\s*\}\}`, regexp.QuoteMeta(tag))
		re := regexp.MustCompile(pat)
		message = re.ReplaceAllString(message, valStr)
	}

	return message
}

func valueToString(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}

	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.CanInterface() {
		return fmt.Sprint(v.Interface())
	}

	return fmt.Sprint(v)
}

type EmailUtilityParam struct {
	Host     string
	Port     int
	Email    string
	Password string
}

func NewEmailUtility(
	param EmailUtilityParam,
) *EmailUtility {
	return &EmailUtility{
		host:     param.Host,
		port:     param.Port,
		email:    param.Email,
		password: param.Password,
		timeout:  1 * time.Minute,
	}
}

type emailBuilder struct {
	to         []string
	subject    string
	message    string
	isHTMLType bool
	host       string
	port       int
	email      string
	password   string
	alias      string
	timeout    time.Duration
}

func (s EmailUtility) NewEmailBuilder() *emailBuilder {
	return &emailBuilder{
		email:    s.email,
		password: s.password,
		host:     s.host,
		port:     s.port,
		alias:    s.alias,
		timeout:  s.timeout,
	}
}

func (s *emailBuilder) Receiver(param ...string) *emailBuilder {
	s.to = param
	return s
}

func (s *emailBuilder) Subject(param string) *emailBuilder {
	s.subject = param
	return s
}

func (s *emailBuilder) Alias(param string) *emailBuilder {
	s.alias = param
	return s
}

func (s *emailBuilder) Message(param string) {
	s.message = param
	go s.send()
}

func (s *emailBuilder) HTMLMessage(param string) {
	s.isHTMLType = true
	s.message = param
	go s.send()
}

func (s *emailBuilder) send() {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Caller().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("Panic recovered")
		}
	}()

	servername := s.host
	if s.port > 0 {
		servername = fmt.Sprintf("%s:%d", s.host, s.port)
	}

	subj := s.subject
	sender := s.email

	if s.alias != "" {
		sender = fmt.Sprintf("%s <%s>", s.alias, s.email)
	}

	headers := make(map[string]string)
	headers["From"] = sender
	headers["To"] = strings.Join(s.to, ",")
	headers["Subject"] = subj
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	content := ""
	for k, v := range headers {
		content += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	if s.isHTMLType {
		content += "mime: MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	}

	content += "\r\n" + s.message
	conn, err := net.DialTimeout("tcp", servername, s.timeout)

	if err != nil {
		log.Error().
			Err(err).
			Str("host", s.host).
			Caller().
			Msg("Error Found when Connect to SMTP Server")
		return
	}

	client, err := smtp.NewClient(conn, servername)
	if err != nil {
		log.Error().
			Err(err).
			Str("server", servername).
			Caller().
			Msg("Error Found when dial to SMTP Server")
		return
	}

	defer client.Quit()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         s.host,
	}

	if err = client.StartTLS(tlsConfig); err != nil {
		log.Error().
			Err(err).
			Interface("config", tlsConfig).
			Caller().
			Msg("Error Found when do TLS to SMTP Server")
		return
	}

	auth := smtp.PlainAuth("", s.email, s.password, servername)
	if err = client.Auth(auth); err != nil {
		log.Error().
			Err(err).
			Interface("auth", auth).
			Caller().
			Msg("Error Found when do auth SMTP Server")
		return
	}

	if err := client.Mail(s.email); err != nil {
		log.Error().
			Err(err).
			Str("mail", s.email).
			Caller().
			Msg("Error Found when set email sender")
		return
	}

	if err := client.Rcpt(strings.Join(s.to, ",")); err != nil {
		log.Error().
			Err(err).
			Interface("receipt", strings.Join(s.to, ",")).
			Caller().
			Msg("Error Found when set email receiver")
		return
	}

	writeCloser, err := client.Data()
	if err != nil {
		log.Error().
			Err(err).
			Caller().
			Msg("Error Found when creating writer")

		return
	}

	defer writeCloser.Close()

	_, err = writeCloser.Write([]byte(content))
	if err != nil {
		log.Error().
			Err(err).
			Caller().
			Msg("Error Found when creating writer")
		return
	}

}
