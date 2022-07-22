package goapstra

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	apiUrlDesignLogicalDevices       = apiUrlDesignPrefix + "logical-devices"
	apiUrlDesignLogicalDevicesPrefix = apiUrlDesignLogicalDevices + apiUrlPathDelim
	apiUrlDesignLogicalDeviceById    = apiUrlDesignLogicalDevicesPrefix + "%s"

	PortIndexingVerticalFirst   = "L-R, T-B"
	PortIndexingHorizontalFirst = "T-B, L-R"
	PortIndexingSchemaAbsolute  = "absolute"
)

type LogicalDevicePortRoleFlags uint16
type logicalDevicePortRole string

const (
	LogicalDevicePortRoleAccess = LogicalDevicePortRoleFlags(1 << iota)
	LogicalDevicePortRoleGeneric
	LogicalDevicePortRoleL3Server
	LogicalDevicePortRoleLeaf
	LogicalDevicePortRolePeer
	LogicalDevicePortRoleServer
	LogicalDevicePortRoleSpine
	LogicalDevicePortRoleSuperspine
	LogicalDevicePortRoleUnused
	LogicalDevicePortRoleUnknown = "unknown logical device port role '%s'"

	logicalDevicePortRoleAccess     = logicalDevicePortRole("access")
	logicalDevicePortRoleGeneric    = logicalDevicePortRole("generic")
	logicalDevicePortRoleL3Server   = logicalDevicePortRole("l3_server")
	logicalDevicePortRoleLeaf       = logicalDevicePortRole("leaf")
	logicalDevicePortRolePeer       = logicalDevicePortRole("peer")
	logicalDevicePortRoleServer     = logicalDevicePortRole("server")
	logicalDevicePortRoleSpine      = logicalDevicePortRole("spine")
	logicalDevicePortRoleSuperspine = logicalDevicePortRole("superspine")
	logicalDevicePortRoleUnused     = logicalDevicePortRole("unused")
)

func (o LogicalDevicePortRoleFlags) raw() []logicalDevicePortRole {
	var result []logicalDevicePortRole
	if o&LogicalDevicePortRoleAccess != 0 {
		result = append(result, logicalDevicePortRoleAccess)
	}
	if o&LogicalDevicePortRoleGeneric != 0 {
		result = append(result, logicalDevicePortRoleGeneric)
	}
	if o&LogicalDevicePortRoleL3Server != 0 {
		result = append(result, logicalDevicePortRoleL3Server)
	}
	if o&LogicalDevicePortRoleLeaf != 0 {
		result = append(result, logicalDevicePortRoleLeaf)
	}
	if o&LogicalDevicePortRolePeer != 0 {
		result = append(result, logicalDevicePortRolePeer)
	}
	if o&LogicalDevicePortRoleServer != 0 {
		result = append(result, logicalDevicePortRoleServer)
	}
	if o&LogicalDevicePortRoleSpine != 0 {
		result = append(result, logicalDevicePortRoleSpine)
	}
	if o&LogicalDevicePortRoleSuperspine != 0 {
		result = append(result, logicalDevicePortRoleSuperspine)
	}
	if o&LogicalDevicePortRoleUnused != 0 {
		result = append(result, logicalDevicePortRoleUnused)
	}
	return result
}

func (o *LogicalDevicePortRoleFlags) Strings() []string {
	var result []string
	for _, role := range o.raw() {
		result = append(result, string(role))
	}
	return result
}

func (o logicalDevicePortRole) parse() (LogicalDevicePortRoleFlags, error) {
	switch o {
	case logicalDevicePortRoleAccess:
		return LogicalDevicePortRoleAccess, nil
	case logicalDevicePortRoleGeneric:
		return LogicalDevicePortRoleGeneric, nil
	case logicalDevicePortRoleL3Server:
		return LogicalDevicePortRoleL3Server, nil
	case logicalDevicePortRoleLeaf:
		return LogicalDevicePortRoleLeaf, nil
	case logicalDevicePortRolePeer:
		return LogicalDevicePortRolePeer, nil
	case logicalDevicePortRoleServer:
		return LogicalDevicePortRoleServer, nil
	case logicalDevicePortRoleSpine:
		return LogicalDevicePortRoleSpine, nil
	case logicalDevicePortRoleSuperspine:
		return LogicalDevicePortRoleSuperspine, nil
	case logicalDevicePortRoleUnused:
		return LogicalDevicePortRoleUnused, nil
	default:
		return 0, fmt.Errorf(LogicalDevicePortRoleUnknown, o)
	}
}

type logicalDevicePortRoles []logicalDevicePortRole

func (o logicalDevicePortRoles) parse() (LogicalDevicePortRoleFlags, error) {
	var result LogicalDevicePortRoleFlags
	for _, r := range o {
		roleFlag, err := r.parse()
		if err != nil {
			return result, err
		}
		result = result | roleFlag
	}
	return result, nil
}

type optionsLogicalDevicesResponse struct {
	Items   []ObjectId `json:"items"`
	Methods []string   `json:"methods"`
}

type getLogicalDevicesResponse struct {
	Items []rawLogicalDevice `json:"items"`
}

type LogicalDevicePanelLayout struct {
	RowCount    int `json:"row_count"`
	ColumnCount int `json:"column_count"`
}

type LogicalDevicePortIndexing struct {
	Order      string `json:"order"`
	StartIndex int    `json:"start_index"`
	Schema     string `json:"schema"` // Valid choices: absolute
}

type LogicalDevicePortGroup struct {
	Count int                        `json:"count"`
	Speed LogicalDevicePortSpeed     `json:"speed"`
	Roles LogicalDevicePortRoleFlags `json:"roles"`
}

func (o LogicalDevicePortGroup) raw() *rawLogicalDevicePortGroup {
	return &rawLogicalDevicePortGroup{
		Count: o.Count,
		Speed: *o.Speed.raw(),
		Roles: o.Roles.raw(),
	}
}

type rawLogicalDevicePortGroup struct {
	Count int                       `json:"count"`
	Speed rawLogicalDevicePortSpeed `json:"speed"`
	Roles logicalDevicePortRoles    `json:"roles"`
}

func (o *rawLogicalDevicePortGroup) parse() (*LogicalDevicePortGroup, error) {
	roles, err := o.Roles.parse()
	if err != nil {
		return nil, err
	}
	return &LogicalDevicePortGroup{
		Count: o.Count,
		Speed: o.Speed.parse(),
		Roles: roles,
	}, nil
}

type LogicalDevicePortSpeed string

func (o LogicalDevicePortSpeed) raw() *rawLogicalDevicePortSpeed {
	if o == "" {
		return nil
	}
	defaultSpeed := rawLogicalDevicePortSpeed{
		Unit:  "G",
		Value: 1,
	}
	lower := strings.ToLower(string(o))
	lower = strings.TrimSpace(lower)
	lower = strings.TrimSuffix(lower, "bps")
	lower = strings.TrimSuffix(lower, "b/s")
	var factor int64
	var trimmed string
	switch {
	case strings.HasSuffix(lower, "m"):
		trimmed = strings.TrimSuffix(lower, "m")
		factor = 1000 * 1000
	case strings.HasSuffix(lower, "g"):
		trimmed = strings.TrimSuffix(lower, "g")
		factor = 1000 * 1000 * 1000
	case strings.HasSuffix(lower, "t"):
		trimmed = strings.TrimSuffix(lower, "t")
		factor = 1000 * 1000 * 1000 * 1000
	default:
		trimmed = lower
		factor = 1
	}
	trimmedInt, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return &defaultSpeed
	}
	bps := trimmedInt * factor
	switch {
	case bps >= 1000*1000*1000: // at least 1Gbps
		return &rawLogicalDevicePortSpeed{
			Unit:  "G",
			Value: int(bps / 1000 / 1000 / 1000),
		}
	case bps >= 10*1000*1000: // at least 10Mbps
		return &rawLogicalDevicePortSpeed{
			Unit:  "M",
			Value: int(bps / 1000 / 1000),
		}
	default:
		return &defaultSpeed
	}
}

func (o *LogicalDevicePortSpeed) BitsPerSecond() int64 {
	return o.raw().BitsPerSecond()
}

type rawLogicalDevicePortSpeed struct {
	Unit  string `json:"unit"`
	Value int    `json:"value"`
}

func (o rawLogicalDevicePortSpeed) parse() LogicalDevicePortSpeed {
	return LogicalDevicePortSpeed(fmt.Sprintf("%d%s", o.Value, o.Unit))
}

func (o *rawLogicalDevicePortSpeed) BitsPerSecond() int64 {
	switch o.Unit {
	case "M":
		return int64(o.Value * 1000 * 1000)
	case "G":
		return int64(o.Value * 1000 * 1000 * 1000)
	default:
		return int64(0)
	}
}

type LogicalDevicePanel struct {
	PanelLayout  LogicalDevicePanelLayout  `json:"panel_layout"`
	PortIndexing LogicalDevicePortIndexing `json:"port_indexing"`
	PortGroups   []LogicalDevicePortGroup  `json:"port_groups"`
}

func (o LogicalDevicePanel) raw() *rawLogicalDevicePanel {
	var portGroups []rawLogicalDevicePortGroup
	for _, pg := range o.PortGroups {
		portGroups = append(portGroups, *pg.raw())
	}

	return &rawLogicalDevicePanel{
		PanelLayout:  o.PanelLayout,
		PortIndexing: o.PortIndexing,
		PortGroups:   portGroups,
	}
}

type rawLogicalDevicePanel struct {
	PanelLayout  LogicalDevicePanelLayout    `json:"panel_layout"`
	PortIndexing LogicalDevicePortIndexing   `json:"port_indexing"`
	PortGroups   []rawLogicalDevicePortGroup `json:"port_groups"`
}

func (o rawLogicalDevicePanel) parse() (*LogicalDevicePanel, error) {
	var portGroups []LogicalDevicePortGroup
	for _, pg := range o.PortGroups {
		p, err := pg.parse()
		if err != nil {
			return nil, err
		}
		portGroups = append(portGroups, *p)
	}

	return &LogicalDevicePanel{
		PanelLayout:  o.PanelLayout,
		PortIndexing: o.PortIndexing,
		PortGroups:   portGroups,
	}, nil
}

type LogicalDevice struct {
	DisplayName    string
	Id             ObjectId
	Panels         []LogicalDevicePanel
	CreatedAt      time.Time
	LastModifiedAt time.Time
}

func (o LogicalDevice) raw() *rawLogicalDevice {
	var panels []rawLogicalDevicePanel
	for _, panel := range o.Panels {
		panels = append(panels, *panel.raw())
	}

	return &rawLogicalDevice{
		DisplayName:    o.DisplayName,
		Id:             o.Id,
		Panels:         panels,
		CreatedAt:      o.CreatedAt,
		LastModifiedAt: o.LastModifiedAt,
	}
}

type rawLogicalDevice struct {
	DisplayName    string                  `json:"display_name"`
	Id             ObjectId                `json:"id,omitempty"`
	Panels         []rawLogicalDevicePanel `json:"panels"`
	CreatedAt      time.Time               `json:"created_at"`
	LastModifiedAt time.Time               `json:"last_modified_at"`
}

func (o rawLogicalDevice) polish() (*LogicalDevice, error) {
	var panels []LogicalDevicePanel
	for _, panel := range o.Panels {
		parsed, err := panel.parse()
		if err != nil {
			return nil, err
		}
		panels = append(panels, *parsed)
	}

	return &LogicalDevice{
		DisplayName:    o.DisplayName,
		Id:             o.Id,
		Panels:         panels,
		CreatedAt:      o.CreatedAt,
		LastModifiedAt: o.LastModifiedAt,
	}, nil
}

func (o *Client) listLogicalDeviceIds(ctx context.Context) ([]ObjectId, error) {
	response := &optionsLogicalDevicesResponse{}
	err := o.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodOptions,
		urlStr:      apiUrlDesignLogicalDevices,
		apiResponse: response,
	})
	if err != nil {
		return nil, convertTtaeToAceWherePossible(err)
	}
	return response.Items, nil
}

func (o *Client) getAllLogicalDevices(ctx context.Context) ([]LogicalDevice, error) {
	response := &getLogicalDevicesResponse{}
	err := o.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodGet,
		urlStr:      apiUrlDesignLogicalDevices,
		apiResponse: response,
	})
	if err != nil {
		return nil, convertTtaeToAceWherePossible(err)
	}
	var result []LogicalDevice
	for _, raw := range response.Items {
		ld, err := raw.polish()
		if err != nil {
			return nil, err
		}
		result = append(result, *ld)
	}
	return result, nil
}

func (o *Client) getLogicalDevice(ctx context.Context, id ObjectId) (*LogicalDevice, error) {
	response := &rawLogicalDevice{}
	err := o.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodGet,
		urlStr:      fmt.Sprintf(apiUrlDesignLogicalDeviceById, id),
		apiResponse: response,
	})
	if err != nil {
		return nil, convertTtaeToAceWherePossible(err)
	}
	return response.polish()
}

func (o *Client) getLogicalDeviceByName(ctx context.Context, name string) (*LogicalDevice, error) {
	logicalDevices, err := o.getAllLogicalDevices(ctx)
	if err != nil {
		return nil, err
	}

	var result LogicalDevice
	var found bool

	for _, ld := range logicalDevices {
		foo := &ld
		_ = foo
		if ld.DisplayName == name {
			if found {
				return nil, ApstraClientErr{
					errType: ErrMultipleMatch,
					err:     fmt.Errorf("found multiple logical devices named '%s' found", name),
				}
			}
			result = ld
			found = true
		}
	}
	if found {
		return &result, nil
	}
	return nil, ApstraClientErr{
		errType: ErrNotfound,
		err:     fmt.Errorf("no logical device named '%s' found", name),
	}
}

func (o *Client) createLogicalDevice(ctx context.Context, in *LogicalDevice) (ObjectId, error) {
	response := &objectIdResponse{}
	err := o.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodPost,
		urlStr:      apiUrlDesignLogicalDevices,
		apiInput:    in.raw(),
		apiResponse: response,
	})
	if err != nil {
		return "", convertTtaeToAceWherePossible(err)
	}
	return response.Id, nil
}

func (o *Client) updateLogicalDevice(ctx context.Context, id ObjectId, in *LogicalDevice) error {
	return o.talkToApstra(ctx, &talkToApstraIn{
		method:   http.MethodPut,
		urlStr:   fmt.Sprintf(apiUrlDesignLogicalDeviceById, id),
		apiInput: in.raw(),
	})
}

func (o *Client) deleteLogicalDevice(ctx context.Context, id ObjectId) error {
	return o.talkToApstra(ctx, &talkToApstraIn{
		method: http.MethodDelete,
		urlStr: fmt.Sprintf(apiUrlDesignLogicalDeviceById, id),
	})
}
