package main

import (
	"testing"
	"time"
)

func TestBot_messageTask(t *testing.T) {
	t.Run("today and tomorrow", func(t *testing.T) {
		data := []FinalSurgeWorkoutData{
			{
				WorkoutDate: "2020-12-23T00:00:00",
				Description: "Warm-up",
			},
			{
				WorkoutDate: "2020-12-23T00:00:00",
				Description: "6 km",
			},
			{
				WorkoutDate: "2020-12-24T00:00:00",
				Description: "12 km",
			},
		}
		expected := `Tasks:
Today 23.12:
Warm-up
6 km

Tomorrow 24.12:
12 km
`

		actual := messageTask(data, time.Date(2020, time.December, 23, 0, 0, 0, 0, time.UTC),
			time.Date(2020, time.December, 24, 0, 0, 0, 0, time.UTC))

		if actual != expected {
			t.Errorf("actual = %s, expected = %s", actual, expected)
		}
	})

	t.Run("today not set", func(t *testing.T) {
		data := []FinalSurgeWorkoutData{
			{
				WorkoutDate: "2020-12-24T00:00:00",
				Description: "12 km",
			},
		}
		expected := `Tasks:
Today 23.12:
not set

Tomorrow 24.12:
12 km
`

		actual := messageTask(data, time.Date(2020, time.December, 23, 0, 0, 0, 0, time.UTC),
			time.Date(2020, time.December, 24, 0, 0, 0, 0, time.UTC))

		if actual != expected {
			t.Errorf("actual = %s, expected = %s", actual, expected)
		}
	})

	t.Run("tomorrow not set", func(t *testing.T) {
		data := []FinalSurgeWorkoutData{
			{
				WorkoutDate: "2020-12-23T00:00:00",
				Description: "Warm-up",
			},
			{
				WorkoutDate: "2020-12-23T00:00:00",
				Description: "6 km",
			},
		}
		expected := `Tasks:
Today 23.12:
Warm-up
6 km

Tomorrow 24.12:
not set
`

		actual := messageTask(data, time.Date(2020, time.December, 23, 0, 0, 0, 0, time.UTC),
			time.Date(2020, time.December, 24, 0, 0, 0, 0, time.UTC))

		if actual != expected {
			t.Errorf("actual = %s, expected = %s", actual, expected)
		}
	})

	t.Run("today and tomorrow not set", func(t *testing.T) {
		var data []FinalSurgeWorkoutData
		expected := `Tasks:
Today 23.12:
not set

Tomorrow 24.12:
not set
`

		actual := messageTask(data, time.Date(2020, time.December, 23, 0, 0, 0, 0, time.UTC),
			time.Date(2020, time.December, 24, 0, 0, 0, 0, time.UTC))

		if actual != expected {
			t.Errorf("actual = %s, expected = %s", actual, expected)
		}
	})
}
