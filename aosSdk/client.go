package aosSdk

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	EnvApstraUser   = "APSTRA_USER"
	EnvApstraPass   = "APSTRA_PASS"
	EnvApstraHost   = "APSTRA_HOST"
	EnvApstraPort   = "APSTRA_PORT"
	EnvApstraScheme = "APSTRA_SCHEME"

	defaultTimeout = 10 * time.Second

	errResponseLimit = 4096

	httpMethodGet    = httpMethod("GET")
	httpMethodPost   = httpMethod("POST")
	httpMethodDelete = httpMethod("DELETE")

	aosAuthHeader = "Authtoken"
)

type httpMethod string

// ClientCfg passed to NewClient() when instantiating a new Client{}
type ClientCfg struct {
	Scheme    string
	Host      string
	Port      uint16
	User      string
	Pass      string
	TlsConfig tls.Config
	Ctx       context.Context
	Timeout   time.Duration
	cancel    func()
}

// Client interacts with an AOS API server
type Client struct {
	baseUrl     string
	cfg         *ClientCfg
	client      *http.Client
	httpHeaders []aosHttpHeader // todo: maybe this should be a dictionary?
	token       *jwt
}

// NewClient creates a Client object
func NewClient(cfg *ClientCfg) *Client {
	if cfg.Ctx == nil {
		cfg.Ctx = context.TODO() // default context
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout // default timeout
	}
	ctxCancel, cancel := context.WithCancel(cfg.Ctx)
	cfg.Ctx = ctxCancel
	cfg.cancel = cancel

	baseUrl := fmt.Sprintf("%s://%s:%d", cfg.Scheme, cfg.Host, cfg.Port)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &cfg.TlsConfig,
		},
	}

	defHdrs := []aosHttpHeader{
		{
			key: "Accept",
			val: "application/json",
		},
	}

	return &Client{cfg: cfg, baseUrl: baseUrl, client: client, httpHeaders: defHdrs}
}

type aosHttpHeader struct {
	key string
	val string
}

type talkToAosIn struct {
	method        httpMethod
	url           string
	toServerPtr   interface{}
	fromServerPtr interface{}
	doNotLogin    bool
}

// talkToAosErr implements error{} and carries around http.Request and
// http.Response object pointers. Error() method produces a string like
// "<error> - http response <status> at url <url>".
// todo: methods like ErrorCRIT() and ErrorWARN()
type talkToAosErr struct {
	request  *http.Request
	response *http.Response
	error    string
}

func (o talkToAosErr) Error() string {
	url := "nil"
	if o.request != nil {
		url = o.request.URL.String()
	}

	status := "nil"
	if o.response != nil {
		status = o.response.Status
	}

	return fmt.Sprintf("%s - http response '%s' at '%s'", o.error, status, url)
}

func (o Client) talkToAos(in *talkToAosIn) error {
	var err error
	var requestBody []byte

	// are we sending data to the server?
	if in.toServerPtr != nil {
		requestBody, err = json.Marshal(in.toServerPtr)
		if err != nil {
			return fmt.Errorf("error marshaling payload in talkToAos for url '%s' - %w", in.url, err)
		}
	}

	// wrap context with timeout
	ctx, cancel := context.WithTimeout(o.cfg.Ctx, o.cfg.Timeout)
	defer cancel()

	// create request
	req, err := http.NewRequestWithContext(ctx, string(in.method), o.baseUrl+in.url, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("error creating http Request for url '%s' - %w", in.url, err)
	}

	// set request httpHeaders
	if in.toServerPtr != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := range o.httpHeaders {
		req.Header.Set(o.httpHeaders[i].key, o.httpHeaders[i].val)
	}

	// talk to the server
	resp, err := o.client.Do(req)
	// trim authentication token from request - Do() has been called - get this out of the way quickly
	req.Header.Del(aosAuthHeader)
	if err != nil { // check error from req.Do()
		return fmt.Errorf("error calling http.client.Do for url '%s' - %w", in.url, err)
	}
	// noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	// response not okay?
	if resp.StatusCode/100 != 2 {

		// Auth fail?
		if resp.StatusCode == 401 {
			// Auth fail at login API is fatal for this transaction
			if in.url == apiUrlUserLogin {
				return newTalkToAosErr(req, requestBody, resp,
					fmt.Sprintf("http %d at '%s' - check username/password",
						resp.StatusCode, in.url))
			}

			// Auth fail with "doNotLogin == true" is fatal for this transaction
			if in.doNotLogin {
				return newTalkToAosErr(req, requestBody, resp,
					fmt.Sprintf("http %d at '%s' and doNotLogin is %t",
						resp.StatusCode, in.url, in.doNotLogin))
			}

			// Try logging in
			err := o.Login()
			if err != nil {
				return fmt.Errorf("error attempting login after initial AuthFail - %w", err)
			}

			// Try the request again
			in.doNotLogin = true
			return o.talkToAos(in)
		} // 401

		return newTalkToAosErr(req, requestBody, resp, "")
	}

	// caller not expecting any response?
	if in.fromServerPtr == nil {
		return nil
	}

	// decode response body into the caller-specified structure
	return json.NewDecoder(resp.Body).Decode(in.fromServerPtr)
}

// newTalkToAosErr returns a talkToAosErr. It's intended to be called after the
// http.Request has been executed with Do(), so the request body has already
// been "spent" by Read(). We'll fill it back in. The response body is likely to
// be closed by a 'defer body.Close()' somewhere, so we'll replace that as well,
// up to some reasonable limit (don't try to buffer gigabytes of data from the
// webserver).
func newTalkToAosErr(req *http.Request, reqBody []byte, resp *http.Response, errMsg string) talkToAosErr {
	url := req.URL.String()
	// don't include secret in error
	req.Header.Del(aosAuthHeader)

	// redact request body for sensitive URLs
	switch url {
	case apiUrlUserLogin:
		req.Body = io.NopCloser(strings.NewReader(fmt.Sprintf("request body for '%s' redacted", url)))
	default:
		rehydratedRequest := bytes.NewBuffer(reqBody)
		req.Body = io.NopCloser(rehydratedRequest)
	}

	// redact response body for sensitive URLs
	switch url {
	case apiUrlUserLogin:
		_ = resp.Body.Close() // close the real network socket
		resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf("resposne body for '%s' redacted", url)))
	default:
		// prepare a stunt double response body for the one that's likely attached to a network
		// socket, and likely to be closed by a `defer` somewhere
		rehydratedResponse := &bytes.Buffer{}
		_, _ = io.CopyN(rehydratedResponse, resp.Body, errResponseLimit) // size limit
		_ = resp.Body.Close()                                            // close the real network socket
		resp.Body = io.NopCloser(rehydratedResponse)                     // replace the original body
	}

	return talkToAosErr{
		request:  req,
		response: resp,
		error:    errMsg,
	}
}

func (o *Client) Login() error {
	var response userLoginResponse
	err := o.talkToAos(&talkToAosIn{
		method: httpMethodPost,
		url:    apiUrlUserLogin,
		toServerPtr: &userLoginRequest{
			Username: o.cfg.User,
			Password: o.cfg.Pass,
		},
		fromServerPtr: &response,
	})
	if err != nil {
		return fmt.Errorf("error talking to AOS in Login - %w", err)
	}

	o.token, err = newJwt(response.Token)
	if err != nil {
		return fmt.Errorf("error parsing JWT in Login - %w", err)
	}

	// stash auth token in client's default set of aos http httpHeaders
	// todo: multiple calls to Login() will cause many Authtoken httpHeaders to be saved
	//  convert to a dictionary or seek/destroy duplicates here
	o.httpHeaders = append(o.httpHeaders, aosHttpHeader{
		key: aosAuthHeader,
		val: o.token.Raw(),
	})

	return nil
}

func (o Client) Logout() error {
	return o.talkToAos(&talkToAosIn{
		method: httpMethodPost,
		url:    apiUrlUserLogout,
	})
}

func (o Client) GetStreamingConfigs() ([]StreamingConfigCfg, error) {
	return o.getAllStreamingConfigs()
}

func (o Client) GetVersion() (*VersionResponse, error) {
	return o.getVersion()
}

// ServerName returns the name of the AOS server this client has been configured to use
func (o Client) ServerName() string {
	return o.cfg.Host
}
