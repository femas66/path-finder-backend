package service

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFCMServiceSimulator(t *testing.T) {
	v := viper.New()
	log := logrus.New()
	log.SetOutput(logrus.StandardLogger().Out)

	fcm := NewFCMService(v, log)
	require.NotNil(t, fcm)
	assert.True(t, fcm.IsSimulator())

	ctx := context.Background()

	t.Run("SendToToken success", func(t *testing.T) {
		err := fcm.SendToToken(ctx, "sample-device-token-12345", NotificationPayload{
			Title: "Test Notification",
			Body:  "Halo dari backend",
			Data: map[string]string{
				"type": "announcement",
			},
		})
		assert.NoError(t, err)
	})

	t.Run("SendToToken empty token error", func(t *testing.T) {
		err := fcm.SendToToken(ctx, "", NotificationPayload{
			Title: "Test",
			Body:  "Test",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token tidak boleh kosong")
	})

	t.Run("SendToTokens success", func(t *testing.T) {
		tokens := []string{"token-1", "token-2", "token-3"}
		success, failure, err := fcm.SendToTokens(ctx, tokens, NotificationPayload{
			Title: "Broadcast",
			Body:  "Notifikasi ke semua pengguna",
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, success)
		assert.Equal(t, 0, failure)
	})

	t.Run("SendToTokens empty tokens list", func(t *testing.T) {
		success, failure, err := fcm.SendToTokens(ctx, []string{}, NotificationPayload{
			Title: "Empty",
			Body:  "Empty",
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, success)
		assert.Equal(t, 0, failure)
	})

	t.Run("SendToTopic success", func(t *testing.T) {
		err := fcm.SendToTopic(ctx, "all_users", NotificationPayload{
			Title: "Topic Notification",
			Body:  "Pesan untuk topik all_users",
		})
		assert.NoError(t, err)
	})

	t.Run("SendToTopic empty topic error", func(t *testing.T) {
		err := fcm.SendToTopic(ctx, "", NotificationPayload{
			Title: "Empty Topic",
			Body:  "Body",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "topic tidak boleh kosong")
	})
}
