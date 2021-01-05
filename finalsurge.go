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
	q := make(url.Values)
	q.Add("request", "login")

	bc, err := json.Marshal(&FinalSurgeCred{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to marshal cred: %w", err)
	}

	bs, err := f.responseBytes(ctx, http.MethodPost, q, nil, bc)
	if err != nil {
		return FinalSurgeLogin{}, fmt.Errorf("failed to get response bytes: %w", err)
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
	q := make(url.Values)
	q.Add("request", "WorkoutList")
	q.Add("scope", "USER")
	q.Add("scopekey", userKey)
	q.Add("startdate", workoutDate(startDate))
	q.Add("enddate", workoutDate(endDate))

	bs, err := f.responseBytes(ctx, http.MethodGet, q, map[string]string{"Authorization": "Bearer " + userToken}, nil)
	if err != nil {
		return FinalSurgeWorkoutList{}, fmt.Errorf("failed to get response bytes: %w", err)
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

func (f *FinalSurgeAPI) responseBytes(ctx context.Context, method string, query url.Values, headers map[string]string,
	body []byte) ([]byte, error) {
	u, err := url.Parse(finalSurgeAPIData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse api data url: %w", err)
	}

	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

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

func workoutDate(t time.Time) string {
	return t.Format("2006-01-02")
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
