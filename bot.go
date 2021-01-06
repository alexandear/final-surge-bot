package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	CommandStart = "start"
)

const (
	KeyboardButtonTask = "/task"
)

type EnterCred int

const (
	EnterUnknown EnterCred = iota
	EnterLogin
	EnterPassword
	EnterDone
)

type UserToken struct {
	UserKey string
	Token   string
}

type Cred struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Sender interface {
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
}

type Storage interface {
	UserToken(ctx context.Context, userName string) (UserToken, error)
	UpdateUserToken(ctx context.Context, userName string, userToken UserToken) error
}

type FinalSurge interface {
	Login(ctx context.Context, email, password string) (FinalSurgeLogin, error)
	Workouts(ctx context.Context, userToken, userKey string, startDate, endDate time.Time,
	) (FinalSurgeWorkoutList, error)
}

type Clock interface {
	Now() time.Time
}

//go:generate mockgen -source=$GOFILE -package main -destination interfaces_mock.go
type Bot struct {
	bot   Sender
	db    Storage
	fs    FinalSurge
	clock Clock

	keyboard tgbotapi.ReplyKeyboardMarkup

	userEnterCreds map[string]EnterCred
	userCreds      map[string]*Cred
}

func NewBot(bot Sender, db Storage, fs FinalSurge, clock Clock) *Bot {
	return &Bot{
		bot:   bot,
		db:    db,
		fs:    fs,
		clock: clock,

		keyboard: tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(KeyboardButtonTask),
		)),

		userEnterCreds: make(map[string]EnterCred),
		userCreds:      make(map[string]*Cred),
	}
}

func (b *Bot) ProcessUpdate(ctx context.Context, update tgbotapi.Update) error {
	if update.Message == nil {
		return nil
	}

	msg, err := b.message(ctx, update.Message)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if msg == nil {
		return nil
	}

	if _, err := b.bot.Send(*msg); err != nil {
		return fmt.Errorf("failed to send reply msg to chat %d: %w", msg.ChatID, err)
	}

	return nil
}

func (b *Bot) message(ctx context.Context, message *tgbotapi.Message) (*tgbotapi.MessageConfig, error) {
	userName := message.From.UserName
	chatID := message.Chat.ID
	text := message.Text

	if message.IsCommand() && message.Command() == CommandStart {
		b.userEnterCreds[userName] = EnterLogin
		b.userCreds[userName] = &Cred{}

		msg := tgbotapi.NewMessage(chatID, "Enter FinalSurge email:")

		return &msg, nil
	}

	if text == KeyboardButtonTask {
		return b.buttonTask(ctx, userName, chatID)
	}

	switch enter := b.userEnterCreds[userName]; enter {
	case EnterUnknown, EnterDone:
	case EnterLogin:
		cred := b.userCreds[userName]
		cred.Email = text

		b.userEnterCreds[userName] = EnterPassword

		msg := tgbotapi.NewMessage(chatID, "Enter FinalSurge password:")

		return &msg, nil
	case EnterPassword:
		cred := b.userCreds[userName]
		cred.Password = text

		b.userEnterCreds[userName] = EnterDone

		login, err := b.fs.Login(ctx, cred.Email, cred.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to login: %w", err)
		}

		userToken := UserToken{
			UserKey: login.Data.UserKey,
			Token:   login.Data.Token,
		}

		if err := b.db.UpdateUserToken(ctx, userName, userToken); err != nil {
			return nil, fmt.Errorf("failed to update user token: %w", err)
		}

		msg := tgbotapi.NewMessage(chatID, "Choose option:")
		msg.ReplyMarkup = b.keyboard

		return &msg, nil
	default:
		log.Printf("unknown enter value %d", enter)
	}

	return nil, nil
}

func (b *Bot) buttonTask(ctx context.Context, userName string, chatID int64) (*tgbotapi.MessageConfig, error) {
	userToken, err := b.db.UserToken(ctx, userName)
	if err != nil {
		return nil, fmt.Errorf("failed to get usertoken: %w", err)
	}

	if userToken.UserKey == "" {
		return nil, nil
	}

	today := newDate(b.clock.Now())
	tomorrow := today.AddDate(0, 0, 1)

	workoutList, err := b.fs.Workouts(context.Background(), userToken.Token, userToken.UserKey, today, tomorrow)
	if err != nil {
		return nil, fmt.Errorf("failed to get workouts: %w", err)
	}

	task := messageTask(workoutList.Data, today, tomorrow)

	msg := tgbotapi.NewMessage(chatID, task)

	return &msg, nil
}

func messageTask(data []FinalSurgeWorkoutData, today, tomorrow time.Time) string {
	todayDescriptions := make([]string, 0, len(data))
	tomorrowDescriptions := make([]string, 0, len(data))

	desc := func(data FinalSurgeWorkoutData) string {
		if IsRestDay(data) {
			return "Rest Day"
		}

		if data.Description != nil {
			return *data.Description
		}

		return ""
	}

	for _, w := range data {
		date, err := time.Parse("2006-01-02T15:04:05", w.WorkoutDate)
		if err != nil {
			log.Printf("failed to parse workout date %s : %v", w.WorkoutDate, err)

			continue
		}

		switch {
		case date.Equal(today):
			todayDescriptions = append(todayDescriptions, desc(w))
		case date.Equal(tomorrow):
			tomorrowDescriptions = append(tomorrowDescriptions, desc(w))
		default:
		}
	}

	task := strings.Builder{}
	task.WriteString("Tasks:")
	task.WriteByte('\n')

	writeDescriptions := func(day string, date time.Time, descriptions []string) {
		task.WriteString(day)
		task.WriteByte(' ')
		task.WriteString(date.Format("02.01"))
		task.WriteByte(':')
		task.WriteByte('\n')

		if len(descriptions) != 0 {
			for _, d := range descriptions {
				task.WriteString(d)
				task.WriteByte('\n')
			}
		} else {
			task.WriteString("not set")
			task.WriteByte('\n')
		}
	}

	writeDescriptions("Today", today, todayDescriptions)
	task.WriteByte('\n')
	writeDescriptions("Tomorrow", tomorrow, tomorrowDescriptions)

	return task.String()
}

func newDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
