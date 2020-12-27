package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	finalSurgeAPIData = "https://beta.finalsurge.com/api/Data"

	activityTypeNameRestDay = "Rest Day"
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
	Data FinalSurgeLoginData `json:"data"`
}

type FinalSurgeLoginData struct {
	UserKey   string `json:"user_key"`
	Token     string `json:"token"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type FinalSurgeWorkoutList struct {
	FinalSurgeStatus
	Data []FinalSurgeWorkoutData `json:"data"`
}

type FinalSurgeWorkoutData struct {
	WorkoutDate string               `json:"workout_date"`
	Description *string              `json:"description"`
	Activities  []FinalSurgeActivity `json:"activities"`
}

type FinalSurgeActivity struct {
	ActivityTypeName string `json:"activity_type_name"`
}

type FinalSurgeStatus struct {
	ServerTime       string  `json:"server_time"`
	Success          bool    `json:"success"`
	ErrorNumber      *int    `json:"error_number"`
	ErrorDescription *string `json:"error_description"`
	CallID           *string `json:"call_id"`
}

func (f *FinalSurgeAPI) Login(ctx context.Context, email, password string) (FinalSurgeLogin, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(bc))
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

	if err := newFinalSurgeError(login.FinalSurgeStatus); err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to get login: %w", err)
	}

	return login, nil
}

func (f *FinalSurgeAPI) Workouts(ctx context.Context, userToken, userKey string, startDate, endDate time.Time,
) (FinalSurgeWorkoutList, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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

	var workoutList FinalSurgeWorkoutList
	if err := json.Unmarshal(bs, &workoutList); err != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to unmarshal: %w", err)
	}

	if err := newFinalSurgeError(workoutList.FinalSurgeStatus); err != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to get workouts: %w", err)
	}

	return workoutList, nil
}

func IsRestDay(data FinalSurgeWorkoutData) bool {
	return len(data.Activities) == 1 && strings.EqualFold(data.Activities[0].ActivityTypeName, activityTypeNameRestDay)
}

func newFinalSurgeError(status FinalSurgeStatus) error {
	if !status.Success && status.ErrorNumber != nil && status.ErrorDescription != nil {
		return fmt.Errorf("final surge error: number=%d desc=%s", *status.ErrorNumber,
			*status.ErrorDescription)
	}

	return nil
}
