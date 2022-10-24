package bot

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestFinalSurgeAPI_Login(t *testing.T) {
	fs := &FinalSurgeAPI{
		client: &http.Client{Timeout: time.Second},
	}

	login := finalSurgeLogin(t, fs)

	t.Logf("%+v", login)
}

func finalSurgeLogin(t *testing.T, fs *FinalSurgeAPI) UserToken {
	email, password := finalSurgeCred()
	if email == "" || password == "" {
		t.Skip()
	}

	login, err := fs.Login(context.Background(), email, password)
	if err != nil {
		t.Fatal(err)
	}

	return login
}

func TestFinalSurgeAPI_Workouts(t *testing.T) {
	fs := &FinalSurgeAPI{
		client: &http.Client{Timeout: time.Second},
	}
	login := finalSurgeLogin(t, fs)

	now := time.Now()
	workouts, err := fs.Workouts(context.Background(), login, now, now.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", workouts)
	for _, w := range workouts {
		t.Logf("workout date: %v", w.Date)
		t.Logf("description: %s", w.Description)
	}
}

func finalSurgeCred() (email, password string) {
	email = os.Getenv("FINAL_SURGE_EMAIL")
	password = os.Getenv("FINAL_SURGE_PASSWORD")

	return email, password
}
