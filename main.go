package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	CommandStart = "start"
	CommandTask  = "task"
)

type EnterCred int

const (
	EnterLogin EnterCred = iota + 1
	EnterPassword
	EnterDone
)

type FinalSurgeCred struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type FinalSurgeStatus struct {
	ServerTime       string  `json:"server_time"`
	Success          bool    `json:"success"`
	ErrorNumber      *int    `json:"error_number"`
	ErrorDescription *string `json:"error_description"`
	CallID           *string `json:"call_id"`
}

type FinalSurgeLogin struct {
	FinalSurgeStatus
	Data struct {
		UserKey   string `json:"user_key"`
		Token     string `json:"token"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	}
}

type FinalSurgeWorkoutList struct {
	FinalSurgeStatus
	Data []struct {
		Description string `json:"description"`
	}
}

func main() {
	apiKey := os.Getenv("BOT_API_KEY")

	bot, err := tgbotapi.NewBotAPI(apiKey)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("authorized on account %s", bot.Self.UserName)

	userEnterCreds := make(map[string]EnterCred)
	userCreds := make(map[string]*FinalSurgeCred)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case CommandStart:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Enter FinalSurge email:")
				if _, err := bot.Send(msg); err != nil {
					log.Printf("error: %v\n", err)
					continue
				}

				userEnterCreds[update.Message.From.UserName] = EnterLogin
				userCreds[update.Message.From.UserName] = &FinalSurgeCred{}
			case CommandTask:
				client := http.Client{Timeout: 2 * time.Second}
				cred := userCreds[update.Message.From.UserName]
				b, err := json.Marshal(&cred)
				if err != nil {
					log.Fatal(err)
				}
				resp, err := client.Post("https://beta.finalsurge.com/api/Data?request=login", "",
					bytes.NewReader(b))
				if err != nil {
					log.Fatal(err)
				}

				bs, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatal(err)
				}
				resp.Body.Close()

				var login FinalSurgeLogin
				if err := json.Unmarshal(bs, &login); err != nil {
					log.Fatal(err)
				}

				if !login.Success && login.ErrorNumber != nil && login.ErrorDescription != nil {
					log.Println("error: ", *login.ErrorNumber, *login.ErrorDescription)
					continue
				}

				log.Println("logged as " + login.Data.FirstName + " " + login.Data.LastName)

				u, err := url.Parse("https://beta.finalsurge.com/api/Data?request=WorkoutList&scope=USER")
				if err != nil {
					log.Fatal()
				}
				q := u.Query()
				q.Set("scopekey", login.Data.UserKey)
				today := time.Now().Format("2006-01-02")
				q.Set("startdate", today)
				q.Set("enddate", today)
				u.RawQuery = q.Encode()

				req, err := http.NewRequest(http.MethodGet, u.String(), nil)
				if err != nil {
					log.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+login.Data.Token)

				resp, err = client.Do(req)
				if err != nil {
					log.Fatal(err)
				}

				bs, err = ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatal(err)
				}
				resp.Body.Close()

				log.Println("raw workouts resp: " + string(bs))

				var workoutList FinalSurgeWorkoutList
				if err := json.Unmarshal(bs, &workoutList); err != nil {
					log.Fatal(err)
				}

				if !workoutList.Success && workoutList.ErrorNumber != nil && workoutList.ErrorDescription != nil {
					log.Println("error: ", *workoutList.ErrorNumber, *workoutList.ErrorDescription)
					continue
				}

				log.Println("today's tasks:")
				for _, w := range workoutList.Data {
					log.Println(w.Description)
				}
			default:
				log.Println("unknown command ", update.Message.Command())
			}

			continue
		}

		switch enter := userEnterCreds[update.Message.From.UserName]; enter {
		case EnterLogin:
			cred := userCreds[update.Message.From.UserName]
			cred.Email = update.Message.Text

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Enter FinalSurge password:")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("error: %v\n", err)
				continue
			}

			userEnterCreds[update.Message.From.UserName] = EnterPassword
		case EnterPassword:
			cred := userCreds[update.Message.From.UserName]
			cred.Password = update.Message.Text

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Got it: %v", userCreds[update.Message.From.UserName]))
			if _, err := bot.Send(msg); err != nil {
				log.Printf("failed to send: %v\n", err)
				continue
			}
			userEnterCreds[update.Message.From.UserName] = EnterDone
		case EnterDone:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Cred exists: %v", userCreds[update.Message.From.UserName]))
			if _, err := bot.Send(msg); err != nil {
				log.Printf("unknown error: %v\n", err)
				continue
			}
		default:
			log.Fatal(fmt.Sprintf("unknown enter value %d", enter))
		}
	}
}
