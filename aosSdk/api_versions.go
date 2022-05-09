package aosSdk

import (
	"fmt"
	"net/url"
)

const (
	apiUrlVersionsPrefix = "/api/versions/"
	apiUrlVersionsAosdi  = apiUrlVersionsPrefix + "aosdi"
	apiUrlVersionsApi    = apiUrlVersionsPrefix + "api"
	apiUrlVersionsBuild  = apiUrlVersionsPrefix + "build"
	apiUrlVersionsDevice = apiUrlVersionsPrefix + "device"
	apiUrlVersionsIba    = apiUrlVersionsPrefix + "iba"
	apiUrlVersionsNode   = apiUrlVersionsPrefix + "node"
	apiUrlVersionsServer = apiUrlVersionsPrefix + "server"
)

type versionsAosdiResponse struct {
	Version       string `json:"version"`
	BuildDateTime string `json:"build_datetime"`
}

type versionsApiResponse struct {
	Major   string `json:"major"`
	Version string `json:"version"`
	Build   string `json:"build"`
	Minor   string `json:"minor"`
}

type versionsBuildResponse struct {
	Version       string `json:"version"`
	BuildDateTime string `json:"build_datetime"`
}

type versionsDeviceRequest struct {
	SerialNumber string `json:"serial_number"`
	Version      string `json:"version"`
	Platform     string `json:"platform"`
}

type versionsDeviceResponse struct {
	Status       string `json:"status"`
	Url          string `json:"url"`
	RetryTimeout int    `json:"retry_timeout"`
	Cksum        string `json:"cksum"`
}

type versionsIbaRequest struct {
	Version  string `json:"version"`
	SystemId string `json:"system_id"`
}

type versionsIbaResponse struct {
	Status       string `json:"status"`
	Url          string `json:"url"`
	RetryTimeout int    `json:"retry_timeout"`
	Cksum        string `json:"cksum"`
}

type versionsNodeRequest struct {
	IpAddress string `json:"ip_address"`
	Version   string `json:"version"`
	SystemId  string `json:"system_id"`
}

type versionsNodeResponse struct {
	Status       string `json:"status"`
	RetryTimeout int    `json:"retry_timeout"`
}

type versionsServerResponse struct {
	Version       string `json:"version"`
	BuildDateTime string `json:"build_datetime"`
}

func (o Client) getVersionsAosdi() (*versionsAosdiResponse, error) {
	url, err := url.Parse(apiUrlVersionsAosdi)
	if err != nil {
		return nil, fmt.Errorf("error parsing url '%s' - %w", apiUrlVersionsAosdi, err)
	}
	var response versionsAosdiResponse
	_, err = o.talkToAos(&talkToAosIn{
		method:        httpMethodGet,
		url:           url,
		fromServerPtr: &response,
	})
	return &response, err
}

func (o Client) getVersionsApi() (*versionsApiResponse, error) {
	url, err := url.Parse(apiUrlVersionsApi)
	if err != nil {
		return nil, fmt.Errorf("error parsing url '%s' - %w", apiUrlVersionsApi, err)
	}
	var response versionsApiResponse
	_, err = o.talkToAos(&talkToAosIn{
		method:        httpMethodGet,
		url:           url,
		fromServerPtr: &response,
	})
	return &response, err
}

func (o Client) getVersionsBuild() (*versionsBuildResponse, error) {
	url, err := url.Parse(apiUrlVersionsBuild)
	if err != nil {
		return nil, fmt.Errorf("error parsing url '%s' - %w", apiUrlVersionsBuild, err)
	}
	var response versionsBuildResponse
	_, err = o.talkToAos(&talkToAosIn{
		method:        httpMethodGet,
		url:           url,
		fromServerPtr: &response,
	})
	return &response, err
}

func (o Client) postVersionsDevice(request *versionsDeviceRequest) (*versionsDeviceResponse, error) {
	url, err := url.Parse(apiUrlVersionsDevice)
	if err != nil {
		return nil, fmt.Errorf("error parsing url '%s' - %w", apiUrlVersionsDevice, err)
	}
	var response versionsDeviceResponse
	_, err = o.talkToAos(&talkToAosIn{
		method:      httpMethodPost,
		url:         url,
		toServerPtr: request,
	})
	return &response, err
}

func (o Client) postVersionsIba(request *versionsIbaRequest) (*versionsIbaResponse, error) {
	url, err := url.Parse(apiUrlVersionsIba)
	if err != nil {
		return nil, fmt.Errorf("error parsing url '%s' - %w", apiUrlVersionsIba, err)
	}
	var response versionsIbaResponse
	_, err = o.talkToAos(&talkToAosIn{
		method:        httpMethodPost,
		url:           url,
		toServerPtr:   request,
		fromServerPtr: &response,
	})
	return &response, err
}

func (o Client) postVersionsNode(request *versionsNodeRequest) (*versionsNodeResponse, error) {
	url, err := url.Parse(apiUrlVersionsNode)
	if err != nil {
		return nil, fmt.Errorf("error parsing url '%s' - %w", apiUrlVersionsNode, err)
	}
	var response versionsNodeResponse
	_, err = o.talkToAos(&talkToAosIn{
		method:        httpMethodPost,
		url:           url,
		toServerPtr:   request,
		fromServerPtr: &response,
	})
	return &response, err
}

func (o Client) getVersionsServer() (*versionsServerResponse, error) {
	url, err := url.Parse(apiUrlVersionsServer)
	if err != nil {
		return nil, fmt.Errorf("error parsing url '%s' - %w", apiUrlVersionsServer, err)
	}
	var response versionsServerResponse
	_, err = o.talkToAos(&talkToAosIn{
		method:        httpMethodGet,
		url:           url,
		fromServerPtr: &response,
	})
	return &response, err
}
