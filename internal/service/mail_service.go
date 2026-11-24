package service

import (
	"fmt"
	"net"
	"net/smtp"
	"time"

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
		defer func() {
			if r := recover(); r != nil {
				s.Log.Errorf("Recovered from panic in SendEmailAsync for %s: %v", to, r)
			}
		}()

		host := s.Viper.GetString("mail.host")
		port := s.Viper.GetInt("mail.port")
		username := s.Viper.GetString("mail.username")
		password := s.Viper.GetString("mail.password")
		from := s.Viper.GetString("mail.from")
		if from == "" {
			from = "no-reply@pathfinder.com"
		}

		if host == "" {
			s.Log.Infof("[MAIL SIMULATOR] To: %s | Subject: %s | Body: [PROTECTED_VERIFICATION_CONTENT]", to, subject)
			return
		}

		msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
			from, to, subject, body)

		addr := fmt.Sprintf("%s:%d", host, port)

		// Gunakan koneksi dengan deadline timeout untuk mencegah goroutine leak jika remote server hang
		dialer := &net.Dialer{Timeout: 15 * time.Second}
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			s.Log.Errorf("Gagal menghubungkan ke SMTP server %s: %v", addr, err)
			return
		}

		client, err := smtp.NewClient(conn, host)
		if err != nil {
			_ = conn.Close()
			s.Log.Errorf("Gagal inisialisasi SMTP client ke %s: %v", addr, err)
			return
		}
		defer client.Close()

		if username != "" && password != "" {
			auth := smtp.PlainAuth("", username, password, host)
			if ok, _ := client.Extension("AUTH"); ok {
				if err := client.Auth(auth); err != nil {
					s.Log.Errorf("Gagal autentikasi SMTP: %v", err)
					return
				}
			}
		}

		if err := client.Mail(from); err != nil {
			s.Log.Errorf("Gagal menetapkan pengirim SMTP: %v", err)
			return
		}
		if err := client.Rcpt(to); err != nil {
			s.Log.Errorf("Gagal menetapkan penerima SMTP: %v", err)
			return
		}

		w, err := client.Data()
		if err != nil {
			s.Log.Errorf("Gagal membuka stream data SMTP: %v", err)
			return
		}

		if _, err := w.Write([]byte(msg)); err != nil {
			_ = w.Close()
			s.Log.Errorf("Gagal menulis data email: %v", err)
			return
		}

		if err := w.Close(); err != nil {
			s.Log.Errorf("Gagal menyelesaikan penulisan email: %v", err)
			return
		}

		_ = client.Quit()
		s.Log.Infof("Email verifikasi berhasil terkirim ke %s", to)
	}()
}
