package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	finalSurgeAPI = "https://beta.finalsurge.com/api"

	activityTypeNameRestDay = "Rest Day"
)

type FinalSurgeAPI struct {
	client *http.Client
}

type FinalSurgeLoginReq struct {
	Email                 string `json:"email"`
	Password              string `json:"password"`
	DeviceManufacturer    string `json:"deviceManufacturer"`
	DeviceModel           string `json:"deviceModel"`
	DeviceOperatingSystem string `json:"deviceOperatingSystem"`
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

func (f *FinalSurgeAPI) Login(ctx context.Context, email, password string) (UserToken, error) {
	bc, err := json.Marshal(&FinalSurgeLoginReq{
		Email:                 email,
		Password:              password,
		DeviceManufacturer:    "",
		DeviceModel:           "Netscape",
		DeviceOperatingSystem: "MacIntel",
	})
	if err != nil {
		return UserToken{}, fmt.Errorf("failed to marshal cred: %w", err)
	}

	h := http.Header{
		"Content-Type": []string{"application/json"},
	}

	bs, err := f.responseBytes(ctx, http.MethodPost, nil, "login", h, bc)
	if err != nil {
		return UserToken{}, fmt.Errorf("failed to get response bytes: %w", err)
	}

	var login FinalSurgeLogin
	if err := json.Unmarshal(bs, &login); err != nil {
		return UserToken{}, fmt.Errorf("failed to unmarshal login: %w", err)
	}

	if err := newFinalSurgeError(login.FinalSurgeStatus); err != nil {
		return UserToken{}, fmt.Errorf("failed to get login: %w", err)
	}

	return UserToken{
		UserKey: login.Data.UserKey,
		Token:   login.Data.Token,
	}, nil
}

func (f *FinalSurgeAPI) Workouts(ctx context.Context, userToken UserToken, startDate, endDate time.Time,
) ([]Workout, error) {
	q := make(url.Values)
	q.Add("scope", "USER")
	q.Add("scopekey", userToken.UserKey)
	q.Add("startdate", finalSurgeDate(startDate))
	q.Add("enddate", finalSurgeDate(endDate))

	header := http.Header{}
	header.Add("Authorization", "Bearer "+userToken.Token)

	bs, err := f.responseBytes(ctx, http.MethodGet, q, "WorkoutList", header, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get response bytes: %w", err)
	}

	var workoutList FinalSurgeWorkoutList
	if err := json.Unmarshal(bs, &workoutList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	if err := newFinalSurgeError(workoutList.FinalSurgeStatus); err != nil {
		return nil, fmt.Errorf("failed to get workouts: %w", err)
	}

	desc := func(data FinalSurgeWorkoutData) string {
		if isRestDay(data) {
			return "Rest Day"
		}

		if data.Description != nil {
			return *data.Description
		}

		return ""
	}

	workouts := make([]Workout, 0, len(workoutList.Data))

	for _, w := range workoutList.Data {
		date, err := time.Parse("2006-01-02T15:04:05", w.WorkoutDate)
		if err != nil {
			log.Printf("failed to parse workout date %s : %v", w.WorkoutDate, err)

			continue
		}

		workouts = append(workouts, Workout{
			Date:        NewDate(date),
			Description: desc(w),
		})
	}

	return workouts, nil
}

func (f *FinalSurgeAPI) responseBytes(ctx context.Context, method string, query url.Values, apiPath string,
	header http.Header, body []byte) ([]byte, error) {
	u, err := url.Parse(finalSurgeAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse api data url: %w", err)
	}

	u.Path = path.Join(u.Path, apiPath)

	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = header

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read all from response body: %w", err)
	}

	if err := resp.Body.Close(); err != nil {
		return nil, fmt.Errorf("failed to close body: %w", err)
	}

	return bs, nil
}

func finalSurgeDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func isRestDay(data FinalSurgeWorkoutData) bool {
	return len(data.Activities) == 1 && strings.EqualFold(data.Activities[0].ActivityTypeName, activityTypeNameRestDay)
}

func newFinalSurgeError(status FinalSurgeStatus) error {
	if !status.Success && status.ErrorNumber != nil && status.ErrorDescription != nil {
		return fmt.Errorf("final surge error: number=%d desc=%s", *status.ErrorNumber,
			*status.ErrorDescription)
	}

	return nil
}
