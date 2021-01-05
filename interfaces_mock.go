// Code generated by MockGen. DO NOT EDIT.
// Source: bot.go

// Package main is a generated GoMock package.
package main

import (
	context "context"
	telegram_bot_api "github.com/go-telegram-bot-api/telegram-bot-api"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
	time "time"
)

// MockSender is a mock of Sender interface
type MockSender struct {
	ctrl     *gomock.Controller
	recorder *MockSenderMockRecorder
}

// MockSenderMockRecorder is the mock recorder for MockSender
type MockSenderMockRecorder struct {
	mock *MockSender
}

// NewMockSender creates a new mock instance
func NewMockSender(ctrl *gomock.Controller) *MockSender {
	mock := &MockSender{ctrl: ctrl}
	mock.recorder = &MockSenderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSender) EXPECT() *MockSenderMockRecorder {
	return m.recorder
}

// Send mocks base method
func (m *MockSender) Send(msg telegram_bot_api.Chattable) (telegram_bot_api.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", msg)
	ret0, _ := ret[0].(telegram_bot_api.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Send indicates an expected call of Send
func (mr *MockSenderMockRecorder) Send(msg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockSender)(nil).Send), msg)
}

// MockStorage is a mock of Storage interface
type MockStorage struct {
	ctrl     *gomock.Controller
	recorder *MockStorageMockRecorder
}

// MockStorageMockRecorder is the mock recorder for MockStorage
type MockStorageMockRecorder struct {
	mock *MockStorage
}

// NewMockStorage creates a new mock instance
func NewMockStorage(ctrl *gomock.Controller) *MockStorage {
	mock := &MockStorage{ctrl: ctrl}
	mock.recorder = &MockStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStorage) EXPECT() *MockStorageMockRecorder {
	return m.recorder
}

// UserToken mocks base method
func (m *MockStorage) UserToken(ctx context.Context, userName string) (UserToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UserToken", ctx, userName)
	ret0, _ := ret[0].(UserToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UserToken indicates an expected call of UserToken
func (mr *MockStorageMockRecorder) UserToken(ctx, userName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UserToken", reflect.TypeOf((*MockStorage)(nil).UserToken), ctx, userName)
}

// UpdateUserToken mocks base method
func (m *MockStorage) UpdateUserToken(ctx context.Context, userName string, userToken UserToken) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUserToken", ctx, userName, userToken)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateUserToken indicates an expected call of UpdateUserToken
func (mr *MockStorageMockRecorder) UpdateUserToken(ctx, userName, userToken interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUserToken", reflect.TypeOf((*MockStorage)(nil).UpdateUserToken), ctx, userName, userToken)
}

// MockFinalSurge is a mock of FinalSurge interface
type MockFinalSurge struct {
	ctrl     *gomock.Controller
	recorder *MockFinalSurgeMockRecorder
}

// MockFinalSurgeMockRecorder is the mock recorder for MockFinalSurge
type MockFinalSurgeMockRecorder struct {
	mock *MockFinalSurge
}

// NewMockFinalSurge creates a new mock instance
func NewMockFinalSurge(ctrl *gomock.Controller) *MockFinalSurge {
	mock := &MockFinalSurge{ctrl: ctrl}
	mock.recorder = &MockFinalSurgeMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFinalSurge) EXPECT() *MockFinalSurgeMockRecorder {
	return m.recorder
}

// Login mocks base method
func (m *MockFinalSurge) Login(ctx context.Context, email, password string) (FinalSurgeLogin, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Login", ctx, email, password)
	ret0, _ := ret[0].(FinalSurgeLogin)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Login indicates an expected call of Login
func (mr *MockFinalSurgeMockRecorder) Login(ctx, email, password interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Login", reflect.TypeOf((*MockFinalSurge)(nil).Login), ctx, email, password)
}

// Workouts mocks base method
func (m *MockFinalSurge) Workouts(ctx context.Context, userToken, userKey string, startDate, endDate time.Time) (FinalSurgeWorkoutList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Workouts", ctx, userToken, userKey, startDate, endDate)
	ret0, _ := ret[0].(FinalSurgeWorkoutList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Workouts indicates an expected call of Workouts
func (mr *MockFinalSurgeMockRecorder) Workouts(ctx, userToken, userKey, startDate, endDate interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Workouts", reflect.TypeOf((*MockFinalSurge)(nil).Workouts), ctx, userToken, userKey, startDate, endDate)
}