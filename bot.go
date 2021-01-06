package main

import (
	"context"
	"fmt"
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

type Cred struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserToken struct {
	UserKey string
	Token   string
}

type Workout struct {
	Date        time.Time
	Description string
}

type Sender interface {
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
}

type Storage interface {
	UserToken(ctx context.Context, userName string) (UserToken, error)
	UpdateUserToken(ctx context.Context, userName string, userToken UserToken) error
}

type FinalSurge interface {
	Login(ctx context.Context, email, password string) (UserToken, error)
	Workouts(ctx context.Context, userToken UserToken, startDate, endDate time.Time) ([]Workout, error)
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

	userCreds map[string]*Cred
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

		userCreds: make(map[string]*Cred),
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
		b.userCreds[userName] = &Cred{}

		msg := tgbotapi.NewMessage(chatID, "Enter FinalSurge email:")

		return &msg, nil
	}

	if text == KeyboardButtonTask {
		return b.buttonTask(ctx, userName, chatID)
	}

	cred, ok := b.userCreds[userName]
	if !ok {
		return nil, nil
	}

	switch {
	case cred.Email == "":
		cred.Email = text

		msg := tgbotapi.NewMessage(chatID, "Enter FinalSurge password:")

		return &msg, nil
	case cred.Email != "":
		cred.Password = text

		userToken, err := b.fs.Login(ctx, cred.Email, cred.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to login: %w", err)
		}

		if err := b.db.UpdateUserToken(ctx, userName, userToken); err != nil {
			return nil, fmt.Errorf("failed to update user token: %w", err)
		}

		msg := tgbotapi.NewMessage(chatID, "Choose option:")
		msg.ReplyMarkup = b.keyboard

		return &msg, nil
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

	today := NewDate(b.clock.Now())
	tomorrow := today.AddDate(0, 0, 1)

	workouts, err := b.fs.Workouts(context.Background(), userToken, today, tomorrow)
	if err != nil {
		return nil, fmt.Errorf("failed to get workouts: %w", err)
	}

	task := messageTask(workouts, today, tomorrow)

	msg := tgbotapi.NewMessage(chatID, task)

	return &msg, nil
}

func messageTask(workouts []Workout, today, tomorrow time.Time) string {
	todayDescriptions := make([]string, 0, len(workouts))
	tomorrowDescriptions := make([]string, 0, len(workouts))

	for _, w := range workouts {
		switch {
		case w.Date.Equal(today):
			todayDescriptions = append(todayDescriptions, w.Description)
		case w.Date.Equal(tomorrow):
			tomorrowDescriptions = append(tomorrowDescriptions, w.Description)
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

func NewDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
