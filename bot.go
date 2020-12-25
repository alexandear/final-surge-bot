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
	CommandTask  = "task"
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

	userEnterCreds map[string]EnterCred
	userCreds      map[string]*Cred
}

func NewBot(bot *tgbotapi.BotAPI, db *Postgres, fs *FinalSurgeAPI) *Bot {
	return &Bot{
		bot: bot,
		db:  db,
		fs:  fs,

		userEnterCreds: make(map[string]EnterCred),
		userCreds:      make(map[string]*Cred),
	}
}

func (b *Bot) Process(ctx context.Context, update tgbotapi.Update) error {
	if update.Message == nil {
		return nil
	}

	userName := update.Message.From.UserName

	if update.Message.IsCommand() {
		return b.command(ctx, update.Message.Command(), userName, update.Message.Chat.ID)
	}

	switch enter := b.userEnterCreds[userName]; enter {
	case EnterLogin:
		cred := b.userCreds[userName]
		cred.Email = update.Message.Text

		text := "Enter FinalSurge password:"
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)

		if _, err := b.bot.Send(msg); err != nil {
			return fmt.Errorf("failed to send msg %s: %w", text, err)
		}

		b.userEnterCreds[userName] = EnterPassword
	case EnterPassword:
		cred := b.userCreds[userName]
		cred.Password = update.Message.Text

		b.userEnterCreds[userName] = EnterDone

		fallthrough
	case EnterUnknown, EnterDone:
		cred := b.userCreds[userName]

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
	default:
		return fmt.Errorf("unknown enter value %d", enter)
	}

	return nil
}

func (b *Bot) command(ctx context.Context, command, userName string, chatID int64) error {
	switch command {
	case CommandStart:
		text := "Enter FinalSurge email:"
		msg := tgbotapi.NewMessage(chatID, text)

		if _, err := b.bot.Send(msg); err != nil {
			return fmt.Errorf("failed to send msg %s: %w", text, err)
		}

		b.userEnterCreds[userName] = EnterLogin
		b.userCreds[userName] = &Cred{}
	case CommandTask:
		userToken, err := b.db.UserToken(ctx, userName)
		if err != nil {
			return fmt.Errorf("failed to get usertoken: %w", err)
		}

		if userToken.UserKey == "" {
			return nil
		}

		today := time.Now()

		workoutList, err := b.fs.Workouts(context.Background(), userToken.Token, userToken.UserKey, today, today)
		if err != nil {
			return fmt.Errorf("failed to get workouts: %w", err)
		}

		text := strings.Builder{}
		text.WriteString("Tasks:")
		text.WriteByte('\n')
		text.WriteString("Today ")
		text.WriteString(today.Format("02.01"))
		text.WriteByte(':')
		text.WriteByte('\n')

		for _, w := range workoutList.Data {
			text.WriteString(w.Description)
			text.WriteByte('\n')
		}

		msg := tgbotapi.NewMessage(chatID, text.String())
		if _, err := b.bot.Send(msg); err != nil {
			return fmt.Errorf("failed to send msg about tasks: %w", err)
		}
	default:
		return fmt.Errorf("unknown command %s", command)
	}

	return nil
}
