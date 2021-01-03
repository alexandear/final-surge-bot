package main

import (
	"context"
	"math/rand"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/golang/mock/gomock"
)

func TestBot_ProcessUpdate(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		senderMock := NewMockSender(ctrl)
		fsMock := NewMockFinalSurge(ctrl)
		storageMock := NewMockStorage(ctrl)
		bot := NewBot(senderMock, storageMock, fsMock)

		const userName = "alexandear"
		chatID := int64(rand.Int())
		entities := []tgbotapi.MessageEntity{
			{Type: "bot_command", Offset: 0, Length: 21},
		}
		const text = "/start@final_surge_bot"
		senderMock.EXPECT().Send(tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID: chatID,
			},
			Text: "Enter FinalSurge email:",
		}).Times(1)

		if err := bot.ProcessUpdate(context.Background(), tgbotapi.Update{
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{
					ID: chatID,
				},
				From: &tgbotapi.User{
					UserName: userName,
				},
				Entities: &entities,
				Text:     text,
			},
		}); err != nil {
			t.Fatalf("start failed: %v", err)
		}
	})
}

func TestBot_messageTask(t *testing.T) {
	for name, tc := range map[string]struct {
		data     []FinalSurgeWorkoutData
		expected string
	}{
		"today and tomorrow": {
			data: []FinalSurgeWorkoutData{
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Description: ptrString("Warm-up"),
				},
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Description: ptrString("6 km"),
				},
				{
					WorkoutDate: "2020-12-24T00:00:00",
					Description: ptrString("12 km"),
				},
			},
			expected: `Tasks:
Today 23.12:
Warm-up
6 km

Tomorrow 24.12:
12 km
`,
		},
		"today not set": {
			data: []FinalSurgeWorkoutData{
				{
					WorkoutDate: "2020-12-24T00:00:00",
					Description: ptrString("12 km"),
				},
			},
			expected: `Tasks:
Today 23.12:
not set

Tomorrow 24.12:
12 km
`,
		},
		"tomorrow not set": {
			data: []FinalSurgeWorkoutData{
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Description: ptrString("Warm-up"),
				},
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Description: ptrString("6 km"),
				},
			},
			expected: `Tasks:
Today 23.12:
Warm-up
6 km

Tomorrow 24.12:
not set
`,
		},
		"today and tomorrow not set": {
			data: []FinalSurgeWorkoutData{},
			expected: `Tasks:
Today 23.12:
not set

Tomorrow 24.12:
not set
`,
		},
		"rest day": {
			data: []FinalSurgeWorkoutData{
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Activities: []FinalSurgeActivity{
						{
							ActivityTypeName: "Rest Day",
						},
					},
				},
				{
					WorkoutDate: "2020-12-24T00:00:00",
					Description: ptrString("12 km"),
				},
			},
			expected: `Tasks:
Today 23.12:
Rest Day

Tomorrow 24.12:
12 km
`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual := messageTask(tc.data, time.Date(2020, time.December, 23, 0, 0, 0, 0, time.UTC),
				time.Date(2020, time.December, 24, 0, 0, 0, 0, time.UTC))

			if actual != tc.expected {
				t.Errorf("actual=%s, expected=%s", actual, tc.expected)
			}
		})
	}
}

func ptrString(s string) *string {
	return &s
}
