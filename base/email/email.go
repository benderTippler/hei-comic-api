package email

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/jordan-wright/email"
	"github.com/labstack/gommon/log"
	"net/smtp"
)

type comicEmail struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
}

const (
	SMTPHost     = "smtp.qq.com"
	SMTPPort     = ":587"
	SMTPUsername = "503186749@qq.com"
	SMTPPassword = "rwwzqwmvrqrmcaja"
)

func NewEmail(ctx context.Context) *comicEmail {
	smtpVar, err := g.Cfg().Get(ctx, "smtp")
	if err != nil {
		log.Fatal(err)
	}
	smtpConf := smtpVar.MapStrStr()
	return &comicEmail{
		smtpHost:     smtpConf["host"],
		smtpPort:     smtpConf["port"],
		smtpUsername: smtpConf["username"],
		smtpPassword: smtpConf["password"],
	}
}

func (c *comicEmail) SendEmail(receiver, subject, text string) error {
	auth := smtp.PlainAuth("", c.smtpUsername, c.smtpPassword, c.smtpHost)
	e := &email.Email{
		From:    fmt.Sprintf("漫画大师<%s>", c.smtpUsername),
		To:      []string{receiver},
		Subject: subject,
		Text:    []byte(text),
	}
	err := e.Send(fmt.Sprintf("%v%v", c.smtpHost, c.smtpPort), auth)
	if err != nil {
		return err
	}
	return nil
}
