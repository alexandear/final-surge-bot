package main

import (
	"context"
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
		bot := NewBot(senderMock, storageMock, fsMock, nil)
		const userName = "alexandear"
		const chatID = int64(20)

		const startCommand = "/start@final_surge_bot"
		senderMock.EXPECT().Send(tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{ChatID: chatID},
			Text:     "Enter FinalSurge email:",
		}).Times(1)
		if err := bot.ProcessUpdate(context.Background(), tgbotapi.Update{
			Message: &tgbotapi.Message{
				Chat:     &tgbotapi.Chat{ID: chatID},
				From:     &tgbotapi.User{UserName: userName},
				Entities: &[]tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: len(startCommand)}},
				Text:     startCommand,
			},
		}); err != nil {
			t.Fatal(err)
		}

		const email = "user@example.com"
		senderMock.EXPECT().Send(tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{ChatID: chatID},
			Text:     "Enter FinalSurge password:",
		}).Times(1)
		if err := bot.ProcessUpdate(context.Background(), tgbotapi.Update{
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
				From: &tgbotapi.User{UserName: userName},
				Text: email,
			},
		}); err != nil {
			t.Fatal(err)
		}

		const password = "password"
		userToken := UserToken{
			UserKey: "b0d1c67e-0d8c-4b67-8faa-c02104ec4f72",
			Token:   "7f2a5f06-1b20-4dde-ba31-2c0a33be6b69",
		}
		fsMock.EXPECT().Login(gomock.Any(), email, password).Return(userToken, nil).Times(1)
		storageMock.EXPECT().UpdateUserToken(gomock.Any(), userName, userToken).Return(nil).Times(1)
		senderMock.EXPECT().Send(tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID:      chatID,
				ReplyMarkup: bot.keyboard,
			},
			Text: "Choose option:",
		}).Times(1)
		if err := bot.ProcessUpdate(context.Background(), tgbotapi.Update{
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
				From: &tgbotapi.User{UserName: userName},
				Text: password,
			},
		}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("token not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		senderMock := NewMockSender(ctrl)
		fsMock := NewMockFinalSurge(ctrl)
		storageMock := NewMockStorage(ctrl)
		bot := NewBot(senderMock, storageMock, fsMock, nil)
		const userName = "alexandear"
		const chatID = int64(20)

		storageMock.EXPECT().UserToken(gomock.Any(), userName).Return(UserToken{}, ErrNotFound).Times(1)
		senderMock.EXPECT().Send(tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID: chatID,
			},
			Text: "Please authorize first by entering /start",
		}).Times(1)
		if err := bot.ProcessUpdate(context.Background(), tgbotapi.Update{
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
				From: &tgbotapi.User{UserName: userName},
				Text: "/task",
			},
		}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("button task", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		senderMock := NewMockSender(ctrl)
		fsMock := NewMockFinalSurge(ctrl)
		storageMock := NewMockStorage(ctrl)
		clockMock := NewMockClock(ctrl)
		bot := NewBot(senderMock, storageMock, fsMock, clockMock)
		const userName = "alexandear"
		const chatID = int64(20)

		userToken := UserToken{
			UserKey: "a0acc35a-c910-4f80-b410-b616d03cf917",
			Token:   "d174c652-b12f-4aad-b730-a43a2c74fa9f",
		}
		now := time.Date(2020, time.December, 20, 15, 15, 20, 0, time.UTC)
		today := time.Date(2020, time.December, 20, 0, 0, 0, 0, time.UTC)
		clockMock.EXPECT().Now().Return(now).Times(1)
		storageMock.EXPECT().UserToken(gomock.Any(), userName).Return(userToken, nil).Times(1)
		fsMock.EXPECT().Workouts(gomock.Any(), userToken,
			today, time.Date(2020, time.December, 21, 0, 0, 0, 0, time.UTC)).
			Return([]Workout{
				{
					Date:        today,
					Description: "10 km",
				},
			}, nil).Times(1)
		senderMock.EXPECT().Send(tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{ChatID: chatID},
			Text: `Tasks:
Today 20.12:
10 km

Tomorrow 21.12:
not set
`,
		}).Times(1)
		if err := bot.ProcessUpdate(context.Background(), tgbotapi.Update{
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: chatID},
				From: &tgbotapi.User{UserName: userName},
				Text: "/task",
			},
		}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestBot_messageTask(t *testing.T) {
	today := time.Date(2020, time.December, 23, 0, 0, 0, 0, time.UTC)
	tomorrow := time.Date(2020, time.December, 24, 0, 0, 0, 0, time.UTC)
	for name, tc := range map[string]struct {
		data     []Workout
		expected string
	}{
		"today and tomorrow": {
			data: []Workout{
				{
					Date:        today,
					Description: "Warm-up",
				},
				{
					Date:        today,
					Description: "6 km",
				},
				{
					Date:        tomorrow,
					Description: "12 km",
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
			data: []Workout{
				{
					Date:        tomorrow,
					Description: "12 km",
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
			data: []Workout{
				{
					Date:        today,
					Description: "Warm-up",
				},
				{
					Date:        today,
					Description: "6 km",
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
			data: []Workout{},
			expected: `Tasks:
Today 23.12:
not set

Tomorrow 24.12:
not set
`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual := messageTask(tc.data, today, tomorrow)

			if actual != tc.expected {
				t.Errorf("actual=%s, expected=%s", actual, tc.expected)
			}
		})
	}
}
