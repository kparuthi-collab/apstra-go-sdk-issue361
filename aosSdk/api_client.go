package aosSdk

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	EnvApstraUser          = "APSTRA_USER"
	EnvApstraPass          = "APSTRA_PASS"
	EnvApstraHost          = "APSTRA_HOST"
	EnvApstraPort          = "APSTRA_PORT"
	EnvApstraScheme        = "APSTRA_SCHEME"
	EnvApstraApiKeyLogFile = "APSTRA_API_TLS_KEYFILE"

	defaultTimeout = 10 * time.Second

	errResponseLimit     = 4096
	taskIdResponeBufSize = 256

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

// TaskId data structure is returned by Apstra for *some* operations, when the
// URL Query String includes `async=full`
type TaskId struct {
	TaskId string `json:"task_id"`
}

func (o TaskId) String() string        { return o.TaskId }
func (o TaskId) Json() ([]byte, error) { return json.Marshal(&o) }

// objectIdResponse is returned by various calls which create an Apstra object
type objectIdResponse struct {
	Id ObjectId `json:"id"`
}

// ObjectId known to Apstra for various objects/resources
type ObjectId string

// Client interacts with an AOS API server
type Client struct {
	baseUrl     string
	cfg         *ClientCfg
	client      *http.Client
	httpHeaders map[string]string
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

	return &Client{cfg: cfg, baseUrl: baseUrl, client: client, httpHeaders: map[string]string{"Accept": "application/json"}}
}

// ServerName returns the name of the AOS server this client has been configured to use
func (o Client) ServerName() string {
	return o.cfg.Host
}

func parseBytesAsTaskId(peek []byte, result *TaskId) bool {
	err := json.Unmarshal(peek, result)
	// wild assumption: every error means "peek doesn't look like a TaskId".
	// there is no error which indicates a problem of any other type.
	if err != nil { // unmarshal fail
		return false
	} else { // good unmarshal, but what about the contents?
		return result.TaskId != ""
	}
}

// Login submits username and password from the ClientCfg (Client.cfg) to the
// Apstra API, retrieves an authorization token. It is optional. If the client
// is not already logged in, Apstra will send HTTP 401. The client will log
// itself in and resubmit the request.
func (o *Client) Login() error {
	return o.login()
}

func (o *Client) login() error {
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

	// stash auth token in client's default set of aos http httpHeaders
	o.httpHeaders[aosAuthHeader] = response.Token

	return nil
}

// Logout invalidates the Apstra API token held by Client
func (o Client) Logout() error {
	return o.logout()
}

func (o Client) logout() error {
	err := o.talkToAos(&talkToAosIn{
		method: httpMethodPost,
		url:    apiUrlUserLogout,
	})
	if err != nil {
		return fmt.Errorf("error calling '%s' - %w", apiUrlUserLogout, err)
	}
	delete(o.httpHeaders, aosAuthHeader)
	return nil
}

// functions below here are implemented in other files.

// GetAllBlueprintIds returns a slice of IDs representing all blueprints
func (o Client) GetAllBlueprintIds() ([]ObjectId, error) {
	return o.getAllBlueprintIds()
}

// GetBlueprint returns *GetBlueprintResponse detailing the requested blueprint
func (o Client) GetBlueprint(in ObjectId) (*GetBlueprintResponse, error) {
	return o.getBlueprint(in)
}

// GetAllStreamingConfigIds returns a slice of ObjectId representing
// currently configured Apstra streaming configs / receivers
func (o Client) GetAllStreamingConfigIds() ([]ObjectId, error) {
	return o.getAllStreamingConfigIds()
}

// GetStreamingConfig returns a slice of *StreamingConfigInfo representing
// the requested Apstra streaming configs / receivers
func (o Client) GetStreamingConfig(id ObjectId) (*StreamingConfigInfo, error) {
	return o.getStreamingConfig(id)
}

// NewStreamingConfig creates a StreamingConfig (Streaming Receiver) on the
// Apstra server.
func (o Client) NewStreamingConfig(cfg *StreamingConfigParams) (ObjectId, error) {
	return o.newStreamingConfig(cfg)
}

// DeleteStreamingConfig deletes the specified streaming config / receiver from
// the Apstra server configuration.
func (o Client) DeleteStreamingConfig(id ObjectId) error {
	return o.deleteStreamingConfig(id)
}

// todo restore this function
//// GetVersion calls apiUrlVersion, returns the Apstra server version as a
//// VersionResponse
//func (o Client) GetVersion() (*VersionResponse, error) {
//	return o.getVersion()
//}

// CreateRoutingZone creates an Apstra Routing Zone / Security Zone / VRF
func (o Client) CreateRoutingZone(cfg *CreateRoutingZoneCfg) (ObjectId, error) {
	return o.createRoutingZone(cfg)
}

func (o Client) DeleteRoutingZone(blueprintId ObjectId, zoneId ObjectId) error {
	return o.deleteRoutingZone(blueprintId, zoneId)
}
