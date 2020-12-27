package main

import (
	"testing"
	"time"
)

func TestBot_messageTask(t *testing.T) {
	for name, tc := range map[string]struct {
		data     []FinalSurgeWorkoutData
		expected string
	}{
		"today and tomorrow": {
			data: []FinalSurgeWorkoutData{
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Description: ptrString("Warm-up"),
				},
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Description: ptrString("6 km"),
				},
				{
					WorkoutDate: "2020-12-24T00:00:00",
					Description: ptrString("12 km"),
				},
			},
			expected: `Tasks:
Today 23.12:
Warm-up
6 km

Tomorrow 24.12:
12 km
`,
		},
		"today not set": {
			data: []FinalSurgeWorkoutData{
				{
					WorkoutDate: "2020-12-24T00:00:00",
					Description: ptrString("12 km"),
				},
			},
			expected: `Tasks:
Today 23.12:
not set

Tomorrow 24.12:
12 km
`,
		},
		"tomorrow not set": {
			data: []FinalSurgeWorkoutData{
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Description: ptrString("Warm-up"),
				},
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Description: ptrString("6 km"),
				},
			},
			expected: `Tasks:
Today 23.12:
Warm-up
6 km

Tomorrow 24.12:
not set
`,
		},
		"today and tomorrow not set": {
			data: []FinalSurgeWorkoutData{},
			expected: `Tasks:
Today 23.12:
not set

Tomorrow 24.12:
not set
`,
		},
		"rest day": {
			data: []FinalSurgeWorkoutData{
				{
					WorkoutDate: "2020-12-23T00:00:00",
					Activities: []FinalSurgeActivity{
						{
							ActivityTypeName: "Rest Day",
						},
					},
				},
				{
					WorkoutDate: "2020-12-24T00:00:00",
					Description: ptrString("12 km"),
				},
			},
			expected: `Tasks:
Today 23.12:
Rest Day

Tomorrow 24.12:
12 km
`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual := messageTask(tc.data, time.Date(2020, time.December, 23, 0, 0, 0, 0, time.UTC),
				time.Date(2020, time.December, 24, 0, 0, 0, 0, time.UTC))

			if actual != tc.expected {
				t.Errorf("actual=%s, expected=%s", actual, tc.expected)
			}
		})
	}
}

func ptrString(s string) *string {
	return &s
}
