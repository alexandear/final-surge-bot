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

type Bot struct {
	bot *tgbotapi.BotAPI
	db  *Postgres
	fs  *FinalSurgeAPI

	keyboard tgbotapi.ReplyKeyboardMarkup

	userEnterCreds map[string]EnterCred
	userCreds      map[string]*Cred
}

func NewBot(bot *tgbotapi.BotAPI, db *Postgres, fs *FinalSurgeAPI) *Bot {
	return &Bot{
		bot: bot,
		db:  db,
		fs:  fs,

		keyboard: tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(KeyboardButtonTask),
		)),

		userEnterCreds: make(map[string]EnterCred),
		userCreds:      make(map[string]*Cred),
	}
}

func (b *Bot) ProcessMessage(ctx context.Context, message *tgbotapi.Message) error {
	userName := message.From.UserName
	chatID := message.Chat.ID
	text := message.Text

	if message.IsCommand() && message.Command() == CommandStart {
		return b.commandStart(ctx, userName, chatID)
	}

	if text == KeyboardButtonTask {
		return b.buttonTask(ctx, userName, chatID)
	}

	switch enter := b.userEnterCreds[userName]; enter {
	case EnterUnknown, EnterDone:
	case EnterLogin:
		cred := b.userCreds[userName]
		cred.Email = text

		msgText := "Enter FinalSurge password:"
		msg := tgbotapi.NewMessage(chatID, msgText)

		if _, err := b.bot.Send(msg); err != nil {
			return fmt.Errorf("failed to send msg %s: %w", msgText, err)
		}

		b.userEnterCreds[userName] = EnterPassword
	case EnterPassword:
		cred := b.userCreds[userName]
		cred.Password = text

		b.userEnterCreds[userName] = EnterDone

		login, err := b.fs.Login(ctx, cred.Email, cred.Password)
		if err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}

		userToken := UserToken{
			UserKey: login.Data.UserKey,
			Token:   login.Data.Token,
		}

		if err := b.db.UpdateUserToken(ctx, userName, userToken); err != nil {
			return fmt.Errorf("failed to update user token: %w", err)
		}

		msg := tgbotapi.NewMessage(chatID, "Choose option:")
		msg.ReplyMarkup = b.keyboard

		if _, err := b.bot.Send(msg); err != nil {
			return fmt.Errorf("failed to set keyboard markup: %w", err)
		}
	default:
		log.Printf("unknown enter value %d", enter)
	}

	return nil
}

func (b *Bot) commandStart(_ context.Context, userName string, chatID int64) error {
	text := "Enter FinalSurge email:"
	msg := tgbotapi.NewMessage(chatID, text)

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send msg %s: %w", text, err)
	}

	b.userEnterCreds[userName] = EnterLogin
	b.userCreds[userName] = &Cred{}

	return nil
}

func (b *Bot) buttonTask(ctx context.Context, userName string, chatID int64) error {
	userToken, err := b.db.UserToken(ctx, userName)
	if err != nil {
		return fmt.Errorf("failed to get usertoken: %w", err)
	}

	if userToken.UserKey == "" {
		return nil
	}

	today := newDate(time.Now())
	tomorrow := today.AddDate(0, 0, 1)

	workoutList, err := b.fs.Workouts(context.Background(), userToken.Token, userToken.UserKey, today, tomorrow)
	if err != nil {
		return fmt.Errorf("failed to get workouts: %w", err)
	}

	task := messageTask(workoutList.Data, today, tomorrow)

	msg := tgbotapi.NewMessage(chatID, task)
	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send msg about tasks: %w", err)
	}

	return nil
}

func messageTask(data []FinalSurgeWorkoutData, today, tomorrow time.Time) string {
	todayDescriptions := make([]string, 0, len(data))
	tomorrowDescriptions := make([]string, 0, len(data))

	for _, w := range data {
		workoutDate, err := time.Parse("2006-01-02T15:04:05", w.WorkoutDate)
		if err != nil {
			log.Printf("failed to parse workout date %s : %v", w.WorkoutDate, err)

			continue
		}

		if workoutDate.Equal(today) {
			todayDescriptions = append(todayDescriptions, w.Description)

			continue
		}

		tomorrowDescriptions = append(tomorrowDescriptions, w.Description)
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
