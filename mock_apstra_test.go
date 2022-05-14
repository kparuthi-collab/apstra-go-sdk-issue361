package goapstra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	mockApstraUser = "mockAdmin"
	mockApstraPass = "mockPassword"
)

type mockApstraApi struct {
	username  string
	password  string
	authToken string
	metricDb  metricdbResponse
}

func newMockApstraApi(password string) (*mockApstraApi, error) {
	var err error
	mock := &mockApstraApi{
		username: mockApstraUser,
	}
	mock.changePassword(password)
	err = mock.createMetricdb()
	if err != nil {
		return nil, err
	}

	return mock, nil
}

func (o *mockApstraApi) changePassword(password string) {
	o.password = password
	o.authToken = randJwt()
}

func (o *mockApstraApi) Do(req *http.Request) (*http.Response, error) {
	switch {
	case req.URL.Path == apiUrlUserLogin:
		return o.handleLogin(req)
	case req.URL.Path == apiUrlUserLogout:
		return o.handleLogout(req)
	case req.URL.Path == apiUrlMetricdbMetric:
		return o.handleMetricdb(req)
	default:
		return nil, fmt.Errorf("mock client doesn't handle API path '%s'", req.URL.Path)
	}
}

func (o *mockApstraApi) handleLogin(req *http.Request) (*http.Response, error) {
	inBody, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body in mockApstraApi.handleLogin() - %w", err)
	}

	in := &userLoginRequest{}
	err = json.Unmarshal(inBody, in)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling userLoginRequest in mockApstraApi.handleLogin() - %w", err)
	}

	if in.Username != o.username || in.Password != o.password {
		return nil, fmt.Errorf("error bad authentication in mockApstraApi.handleLogin() '%s:%s' vs. '%s:%s",
			in.Username, in.Password, o.username, o.password)
	}
	o.authToken = randJwt()
	outBody, err := json.Marshal(userLoginResponse{
		Token: o.authToken,
		Id:    randId(),
	})

	return &http.Response{
		Body:       io.NopCloser(bytes.NewReader(outBody)),
		StatusCode: http.StatusCreated,
	}, nil
}

func (o mockApstraApi) handleLogout(req *http.Request) (*http.Response, error) {
	for _, val := range req.Header.Values(apstraAuthHeader) {
		if val == o.authToken {
			o.authToken = ""
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		}
	}
	return nil, fmt.Errorf("logout attempt without valid token in mockApstraApi.handleLogin()")
}

func (o mockApstraApi) handleMetricdb(req *http.Request) (*http.Response, error) {
	var authOk bool
authLoop:
	for _, val := range req.Header.Values(apstraAuthHeader) {
		if val == o.authToken {
			authOk = true
			break authLoop
		}
	}
	if authOk == false {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	}

	outBody, err := json.Marshal(o.metricDb)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		Body:       io.NopCloser(bytes.NewReader(outBody)),
		StatusCode: http.StatusOK,
	}, nil
}