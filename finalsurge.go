package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	finalSurgeAPIData = "https://beta.finalsurge.com/api/Data"
)

type FinalSurgeAPI struct {
	client *http.Client
}

type FinalSurgeCred struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

type FinalSurgeStatus struct {
	ServerTime       string  `json:"server_time"`
	Success          bool    `json:"success"`
	ErrorNumber      *int    `json:"error_number"`
	ErrorDescription *string `json:"error_description"`
	CallID           *string `json:"call_id"`
}

func NewFinalSurgeAPI(client *http.Client) *FinalSurgeAPI {
	return &FinalSurgeAPI{
		client: client,
	}
}

func (f *FinalSurgeAPI) Login(email, password string) (FinalSurgeLogin, error) {
	u, err := url.Parse(finalSurgeAPIData)
	if err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to parse final surge api data url: %w", err)
	}

	q := u.Query()
	q.Set("request", "login")
	u.RawQuery = q.Encode()

	bc, err := json.Marshal(&FinalSurgeCred{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to marshal cred: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(bc))
	if err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to do request: %w", err)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to read all: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to close body: %w", err)
	}

	var login FinalSurgeLogin
	if err := json.Unmarshal(bs, &login); err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to unmarshal login: %w", err)
	}

	if !login.Success && login.ErrorNumber != nil && login.ErrorDescription != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to get login: %d %s", *login.ErrorNumber, *login.ErrorDescription)
	}

	return login, nil
}

func (f *FinalSurgeAPI) Workouts(userToken, userKey string, startDate, endDate time.Time) (FinalSurgeWorkoutList, error) {
	u, err := url.Parse(finalSurgeAPIData)
	if err != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to parse final surge api data url: %w", err)
	}

	q := u.Query()
	q.Set("request", "WorkoutList")
	q.Set("scope", "USER")
	q.Set("scopekey", userKey)
	q.Set("startdate", startDate.Format("2006-01-02"))
	q.Set("enddate", endDate.Format("2006-01-02"))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+userToken)

	resp, err := f.client.Do(req)
	if err != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to do request: %w", err)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to read all: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to close body: %w", err)
	}

	log.Println("raw workouts resp: " + string(bs))

	var workoutList FinalSurgeWorkoutList
	if err := json.Unmarshal(bs, &workoutList); err != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to unmarshal: %w", err)
	}

	if !workoutList.Success && workoutList.ErrorNumber != nil && workoutList.ErrorDescription != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to get workout list: %d %s", *workoutList.ErrorNumber, *workoutList.ErrorDescription)
	}

	return workoutList, nil
}