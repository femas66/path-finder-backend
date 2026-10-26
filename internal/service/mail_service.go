package service

import (
	"fmt"
	"net/smtp"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type MailService interface {
	SendEmailAsync(to string, subject string, body string)
}

type mailServiceImpl struct {
	Viper *viper.Viper
	Log   *logrus.Logger
}

func NewMailService(viperConfig *viper.Viper, logger *logrus.Logger) MailService {
	return &mailServiceImpl{
		Viper: viperConfig,
		Log:   logger,
	}
}

func (s *mailServiceImpl) SendEmailAsync(to string, subject string, body string) {
	// Jalankan secara asynchronous menggunakan goroutine agar respons API instan (< 10ms)
	go func() {
		host := s.Viper.GetString("mail.host")
		port := s.Viper.GetInt("mail.port")
		username := s.Viper.GetString("mail.username")
		password := s.Viper.GetString("mail.password")
		from := s.Viper.GetString("mail.from")
		if from == "" {
			from = "no-reply@pathfinder.com"
		}

		if host == "" {
			s.Log.Infof("[MAIL SIMULATOR] To: %s | Subject: %s | Body: %s", to, subject, body)
			return
		}

		msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
			from, to, subject, body)

		addr := fmt.Sprintf("%s:%d", host, port)
		auth := smtp.PlainAuth("", username, password, host)

		err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
		if err != nil {
			s.Log.Errorf("Gagal mengirim email ke %s: %v", to, err)
			return
		}

		s.Log.Infof("Email verifikasi berhasil terkirim ke %s", to)
	}()
}
