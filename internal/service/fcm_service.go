package service

import (
	"context"
	"errors"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

type NotificationPayload struct {
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	ImageURL string            `json:"image_url,omitempty"`
	Data     map[string]string `json:"data,omitempty"`
}

type FCMService interface {
	SendToToken(ctx context.Context, token string, payload NotificationPayload) error
	SendToTokens(ctx context.Context, tokens []string, payload NotificationPayload) (successCount int, failureCount int, err error)
	SendToTopic(ctx context.Context, topic string, payload NotificationPayload) error
	IsSimulator() bool
}

type fcmServiceImpl struct {
	Viper       *viper.Viper
	Log         *logrus.Logger
	client      *messaging.Client
	isSimulator bool
}

func NewFCMService(viperConfig *viper.Viper, logger *logrus.Logger) FCMService {
	credentialsFile := viperConfig.GetString("firebase.credentials_file")
	credentialsJSON := viperConfig.GetString("firebase.credentials_json")
	projectID := viperConfig.GetString("firebase.project_id")

	var opts []option.ClientOption

	if credentialsJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credentialsJSON)))
	} else if credentialsFile != "" {
		if _, err := os.Stat(credentialsFile); err == nil {
			opts = append(opts, option.WithCredentialsFile(credentialsFile))
		} else {
			logger.Warnf("[FCM] File kredensial %s tidak ditemukan, beralih ke mode SIMULATOR", credentialsFile)
		}
	}

	var fbConfig *firebase.Config
	if projectID != "" {
		fbConfig = &firebase.Config{ProjectID: projectID}
	}

	if len(opts) == 0 {
		logger.Info("[FCM] Berjalan dalam mode SIMULATOR (kredensial Firebase belum dikonfigurasi)")
		return &fcmServiceImpl{
			Viper:       viperConfig,
			Log:         logger,
			client:      nil,
			isSimulator: true,
		}
	}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, fbConfig, opts...)
	if err != nil {
		logger.Errorf("[FCM] Gagal inisialisasi Firebase App: %v, beralih ke mode SIMULATOR", err)
		return &fcmServiceImpl{
			Viper:       viperConfig,
			Log:         logger,
			client:      nil,
			isSimulator: true,
		}
	}

	messagingClient, err := app.Messaging(ctx)
	if err != nil {
		logger.Errorf("[FCM] Gagal membuat Firebase Messaging Client: %v, beralih ke mode SIMULATOR", err)
		return &fcmServiceImpl{
			Viper:       viperConfig,
			Log:         logger,
			client:      nil,
			isSimulator: true,
		}
	}

	logger.Info("[FCM] Firebase Cloud Messaging berhasil diinisialisasi")
	return &fcmServiceImpl{
		Viper:       viperConfig,
		Log:         logger,
		client:      messagingClient,
		isSimulator: false,
	}
}

func (s *fcmServiceImpl) IsSimulator() bool {
	return s.isSimulator
}

func (s *fcmServiceImpl) SendToToken(ctx context.Context, token string, payload NotificationPayload) error {
	if token == "" {
		return errors.New("token tidak boleh kosong")
	}

	if s.isSimulator || s.client == nil {
		s.Log.Infof("[FCM SIMULATOR] SendToToken -> Token: %s | Title: %s | Body: %s | Data: %+v", token, payload.Title, payload.Body, payload.Data)
		return nil
	}

	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title:    payload.Title,
			Body:     payload.Body,
			ImageURL: payload.ImageURL,
		},
		Data: payload.Data,
	}

	res, err := s.client.Send(ctx, msg)
	if err != nil {
		s.Log.Errorf("[FCM] Gagal kirim notifikasi ke token %s: %v", token, err)
		return err
	}

	s.Log.Infof("[FCM] Notifikasi berhasil dikirim. MessageID: %s", res)
	return nil
}

func (s *fcmServiceImpl) SendToTokens(ctx context.Context, tokens []string, payload NotificationPayload) (int, int, error) {
	if len(tokens) == 0 {
		return 0, 0, nil
	}

	if s.isSimulator || s.client == nil {
		s.Log.Infof("[FCM SIMULATOR] SendToTokens -> Count: %d | Title: %s | Body: %s | Data: %+v", len(tokens), payload.Title, payload.Body, payload.Data)
		return len(tokens), 0, nil
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title:    payload.Title,
			Body:     payload.Body,
			ImageURL: payload.ImageURL,
		},
		Data: payload.Data,
	}

	res, err := s.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		s.Log.Errorf("[FCM] Gagal broadcast notifikasi ke %d tokens: %v", len(tokens), err)
		return 0, len(tokens), err
	}

	s.Log.Infof("[FCM] Multicast selesai. Sukses: %d, Gagal: %d", res.SuccessCount, res.FailureCount)
	return res.SuccessCount, res.FailureCount, nil
}

func (s *fcmServiceImpl) SendToTopic(ctx context.Context, topic string, payload NotificationPayload) error {
	if topic == "" {
		return errors.New("topic tidak boleh kosong")
	}

	if s.isSimulator || s.client == nil {
		s.Log.Infof("[FCM SIMULATOR] SendToTopic -> Topic: %s | Title: %s | Body: %s | Data: %+v", topic, payload.Title, payload.Body, payload.Data)
		return nil
	}

	msg := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title:    payload.Title,
			Body:     payload.Body,
			ImageURL: payload.ImageURL,
		},
		Data: payload.Data,
	}

	res, err := s.client.Send(ctx, msg)
	if err != nil {
		s.Log.Errorf("[FCM] Gagal kirim notifikasi ke topik %s: %v", topic, err)
		return err
	}

	s.Log.Infof("[FCM] Notifikasi topik %s berhasil dikirim. MessageID: %s", topic, res)
	return nil
}
