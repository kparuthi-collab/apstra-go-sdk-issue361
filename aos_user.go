package apstraTelemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	schemeHttp        = "http"
	schemeHttps       = "https"
	schemeHttpsUnsafe = "hxxps"

	aosApiUserLogin  = "/api/user/login"
	aosApiUserLogout = "/api/user/logout"
)

// aosUserLoginRequest payload to the aosApiUserLogin API endpoint
type aosUserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// aosUserLoginResponse payload returned by the aosApiUserLogin API endpoint
type aosUserLoginResponse struct {
	Token string `json:"token"`
	Id    string `json:"id"`
}

// UserLogin submits credentials to an API server, collects a login token
// todo - need to handle token timeout
func (o *AosClient) UserLogin() (err error) {
	msg, err := json.Marshal(aosUserLoginRequest{
		Username: o.cfg.User,
		Password: o.cfg.Pass,
	})
	if err != nil {
		return fmt.Errorf("error marshaling aosLogin object - %v", err)
	}

	req, err := http.NewRequest("POST", o.baseUrl+aosApiUserLogin, bytes.NewBuffer(msg))
	if err != nil {
		return fmt.Errorf("error creating http Request - %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("error calling http.client.Do - %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("http response code is not '%d' got '%d' at '%s'", 201, resp.StatusCode, aosApiUserLogin)
	}

	var loginResp *aosUserLoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		return fmt.Errorf("error decoding aosUserLoginResponse JSON - %v", err)
	}

	o.token = loginResp.Token

	return nil
}

func (o AosClient) UserLogout() error {
	req, err := http.NewRequest("POST", o.baseUrl+aosApiUserLogout, nil)
	if err != nil {
		return fmt.Errorf("error creating http Request - %v", err)
	}
	req.Header.Set("Authtoken", o.token)

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("error calling http.client.Do - %v", err)
	}
	err = resp.Body.Close()
	if err != nil {
		return fmt.Errorf("error closing logout http response body - %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("http response code is not '%d' got '%d' at '%s'", 200, resp.StatusCode, aosApiUserLogout)
	}

	return nil
}
