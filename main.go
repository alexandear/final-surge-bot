package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	bolt "go.etcd.io/bbolt"
)

const (
	DatabaseFile = "final-surge-bot.db"

	BucketUserToken = "UserToken"
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

type UserToken struct {
	UserKey string
	Token   string
}

func main() {
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		log.Fatal("PUBLIC_URL env is missing")
	}

	apiKey := os.Getenv("BOT_API_KEY")
	if apiKey == "" {
		log.Fatal("BOT_API_KEY env is missing")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT env is missing")
	}

	bot, err := tgbotapi.NewBotAPI(apiKey)
	if err != nil {
		log.Panic(fmt.Errorf("failed to init bot api: %w", err))
	}

	db, err := bolt.Open(DatabaseFile, 0o600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to open database: %w", err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println(fmt.Errorf("failed to close db: %w", err))
		}
	}()

	log.Printf("bot authorized on account %s", bot.Self.UserName)

	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(BucketUserToken))
		if errors.Is(err, bolt.ErrBucketExists) {
			return nil
		}
		return err
	}); err != nil {
		log.Fatal(fmt.Errorf("failed to create bucket: %w", err))
	}

	userEnterCreds := make(map[string]EnterCred)
	userCreds := make(map[string]*FinalSurgeCred)

	u := tgbotapi.NewUpdate(0)

	u.Timeout = 60

	if _, err := bot.SetWebhook(tgbotapi.NewWebhook(publicURL + bot.Token)); err != nil {
		log.Fatal(fmt.Errorf("failed to set webhook to %s: %w", publicURL, err))
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get webhook info: %w", err))
	}

	if info.LastErrorDate != 0 {
		log.Printf("telegram callback failed: %s", info.LastErrorMessage)
	}

	updates := bot.ListenForWebhook("/" + bot.Token)

	go func() {
		addr := "0.0.0.0:" + port
		log.Printf("start listening on %s", addr)

		if err := http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})); err != nil {
			log.Println(fmt.Errorf("failed to listen and serve: %w", err))
		}
	}()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		userName := update.Message.From.UserName

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case CommandStart:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Enter FinalSurge email:")
				if _, err := bot.Send(msg); err != nil {
					log.Println(fmt.Errorf("failed to send msg about email: %w", err))
					continue
				}

				userEnterCreds[userName] = EnterLogin
				userCreds[userName] = &FinalSurgeCred{}
			case CommandTask:
				var userToken UserToken
				if err := db.View(func(tx *bolt.Tx) error {
					b := tx.Bucket([]byte(BucketUserToken))
					bc := b.Get([]byte(userName))
					if bc == nil {
						log.Println("token not found for user: ", userName)
						return nil
					}

					if err := json.Unmarshal(bc, &userToken); err != nil {
						return fmt.Errorf("failed to unmarshal usertoken: %w", err)
					}
					return nil
				}); err != nil {
					log.Fatal(fmt.Errorf("failed to get user token: %w", err))
				}

				if userToken.UserKey == "" {
					continue
				}

				log.Println("logged as " + userToken.UserKey)

				u, err := url.Parse("https://beta.finalsurge.com/api/Data?request=WorkoutList&scope=USER")
				if err != nil {
					log.Panic(fmt.Errorf("failed to parse url: %w", err))
				}
				q := u.Query()
				q.Set("scopekey", userToken.UserKey)
				now := time.Now()
				today := now.Format("2006-01-02")
				q.Set("startdate", today)
				q.Set("enddate", today)
				u.RawQuery = q.Encode()

				req, err := http.NewRequest(http.MethodGet, u.String(), nil)
				if err != nil {
					log.Fatal(fmt.Errorf("failed to create request: %w", err))
				}
				req.Header.Set("Authorization", "Bearer "+userToken.Token)

				client := http.Client{Timeout: 2 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					log.Fatal(fmt.Errorf("failed to do request: %w", err))
				}

				bs, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatal(fmt.Errorf("failed to read all: %w", err))
				}
				if err := resp.Body.Close(); err != nil {
					log.Fatal(fmt.Errorf("failed to close body: %w", err))
				}

				log.Println("raw workouts resp: " + string(bs))

				var workoutList FinalSurgeWorkoutList
				if err := json.Unmarshal(bs, &workoutList); err != nil {
					log.Fatal(err)
				}

				if !workoutList.Success && workoutList.ErrorNumber != nil && workoutList.ErrorDescription != nil {
					log.Println(fmt.Errorf("failed to get workout list: %d %s", *workoutList.ErrorNumber, *workoutList.ErrorDescription))
					continue
				}

				text := strings.Builder{}
				text.WriteString("Tasks:")
				text.WriteByte('\n')
				text.WriteString("Today ")
				text.WriteString(now.Format("02.01"))
				text.WriteByte(':')
				text.WriteByte('\n')
				for _, w := range workoutList.Data {
					text.WriteString(w.Description)
					text.WriteByte('\n')
				}

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text.String())
				if _, err := bot.Send(msg); err != nil {
					log.Println(fmt.Errorf("failed to send msg about tasks: %w", err))
				}
			default:
				log.Println("unknown command ", update.Message.Command())
			}

			continue
		}

		switch enter := userEnterCreds[userName]; enter {
		case EnterLogin:
			cred := userCreds[userName]
			cred.Email = update.Message.Text

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Enter FinalSurge password:")
			if _, err := bot.Send(msg); err != nil {
				log.Println(fmt.Errorf("failed to send msg about password: %w", err))
				continue
			}

			userEnterCreds[userName] = EnterPassword
		case EnterPassword:
			cred := userCreds[userName]
			cred.Password = update.Message.Text

			userEnterCreds[userName] = EnterDone
			fallthrough
		case EnterUnknown, EnterDone:
			cred := userCreds[userName]

			b, err := json.Marshal(&cred)
			if err != nil {
				log.Fatal(fmt.Errorf("failed to marshal cred: %w", err))
			}

			client := http.Client{Timeout: 2 * time.Second}
			resp, err := client.Post("https://beta.finalsurge.com/api/Data?request=login", "",
				bytes.NewReader(b))
			if err != nil {
				log.Fatal(fmt.Errorf("failed to do post request: %w", err))
			}

			bs, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(fmt.Errorf("failed to read all: %w", err))
			}
			if err := resp.Body.Close(); err != nil {
				log.Fatal(fmt.Errorf("failed to close body: %w", err))
			}

			var login FinalSurgeLogin
			if err := json.Unmarshal(bs, &login); err != nil {
				log.Fatal(fmt.Errorf("failed to unmarshal login: %w", err))
			}

			if !login.Success && login.ErrorNumber != nil && login.ErrorDescription != nil {
				log.Fatal(fmt.Errorf("failed to get login data: %d %s", *login.ErrorNumber, *login.ErrorDescription))
			}

			userToken := &UserToken{
				UserKey: login.Data.UserKey,
				Token:   login.Data.Token,
			}

			bu, err := json.Marshal(userToken)
			if err != nil {
				log.Fatal(fmt.Errorf("failed to marshal user token: %w", err))
			}

			if err := db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(BucketUserToken))
				return b.Put([]byte(userName), bu)
			}); err != nil {
				log.Fatal(fmt.Errorf("failed to put user token: %w", err))
			}
		default:
			log.Fatal(fmt.Sprintf("unknown enter value %d", enter))
		}
	}
}
