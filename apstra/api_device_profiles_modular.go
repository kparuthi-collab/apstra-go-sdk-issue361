package apstra

import (
	"context"
	"fmt"
	"github.com/orsinium-labs/enum"
	"net/http"
)

type DeviceProfileType enum.Member[string]

func (o DeviceProfileType) String() string {
	return o.Value
}

func (o *DeviceProfileType) FromString(s string) error {
	t := DeviceProfileTypes.Parse(s)
	if t == nil {
		return fmt.Errorf("failed to parse DeviceProfileType %q", s)
	}
	o.Value = t.Value
	return nil
}

var (
	DeviceProfileTypeModular    = DeviceProfileType{Value: "modular"}
	DeviceProfileTypeMonolithic = DeviceProfileType{Value: "monolithic"}
	DeviceProfileTypes          = enum.New(DeviceProfileTypeModular, DeviceProfileTypeMonolithic)
)

type rawModularDeviceSlotConfiguration struct {
	LinecardProfileId ObjectId `json:"linecard_profile_id"`
	SlotId            uint64   `json:"slot_id"`
}

type ModularDeviceSlotConfiguration struct {
	LinecardProfileId ObjectId
}

type ModularDeviceProfile struct {
	Label              string
	ChassisProfileId   ObjectId
	SlotConfigurations map[uint64]ModularDeviceSlotConfiguration
}

func (o *ModularDeviceProfile) raw() *rawModularDeviceProfile {
	result := &rawModularDeviceProfile{
		DeviceProfileType:  DeviceProfileTypeModular.Value,
		Label:              o.Label,
		ChassisProfileId:   o.ChassisProfileId,
		SlotConfigurations: make([]rawModularDeviceSlotConfiguration, len(o.SlotConfigurations)),
	}

	var i int
	for slotId, slotConfiguration := range o.SlotConfigurations {
		result.SlotConfigurations[i] = rawModularDeviceSlotConfiguration{
			SlotId:            slotId,
			LinecardProfileId: slotConfiguration.LinecardProfileId,
		}
		i++
	}

	return result
}

type rawModularDeviceProfile struct {
	DeviceProfileType  string                              `json:"device_profile_type"`
	Label              string                              `json:"label"`
	ChassisProfileId   ObjectId                            `json:"chassis_profile_id"`
	SlotConfigurations []rawModularDeviceSlotConfiguration `json:"slot_configuration"`
}

func (o *rawModularDeviceProfile) polish() *ModularDeviceProfile {
	result := &ModularDeviceProfile{
		Label:              o.Label,
		ChassisProfileId:   o.ChassisProfileId,
		SlotConfigurations: make(map[uint64]ModularDeviceSlotConfiguration, len(o.SlotConfigurations)),
	}

	for _, slotConfiguration := range o.SlotConfigurations {
		result.SlotConfigurations[slotConfiguration.SlotId] = ModularDeviceSlotConfiguration{
			LinecardProfileId: slotConfiguration.LinecardProfileId,
		}
	}

	return result
}

func (o *Client) createModularDeviceProfile(ctx context.Context, in *rawModularDeviceProfile) (ObjectId, error) {
	response := new(objectIdResponse)
	err := o.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodPost,
		urlStr:      apiUrlDeviceProfiles,
		apiInput:    in,
		apiResponse: response,
	})
	if err != nil {
		return "", convertTtaeToAceWherePossible(err)
	}

	return response.Id, nil
}

func (o *Client) getModularDeviceProfile(ctx context.Context, id ObjectId) (*rawModularDeviceProfile, error) {
	response := new(rawModularDeviceProfile)
	err := o.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodGet,
		urlStr:      fmt.Sprintf(apiUrlDeviceProfileById, id),
		apiResponse: response,
	})
	if err != nil {
		return nil, convertTtaeToAceWherePossible(err)
	}

	return response, nil
}

func (o *Client) updateModularDeviceProfile(ctx context.Context, id ObjectId, cfg *rawModularDeviceProfile) error {
	err := o.talkToApstra(ctx, &talkToApstraIn{
		method:   http.MethodPut,
		urlStr:   fmt.Sprintf(apiUrlDeviceProfileById, id),
		apiInput: cfg,
	})
	if err != nil {
		return convertTtaeToAceWherePossible(err)
	}

	return nil
}

func (o *Client) deleteModularDeviceProfile(ctx context.Context, id ObjectId) error {
	err := o.talkToApstra(ctx, &talkToApstraIn{
		method: http.MethodDelete,
		urlStr: fmt.Sprintf(apiUrlDeviceProfileById, id),
	})
	if err != nil {
		return convertTtaeToAceWherePossible(err)
	}

	return nil
}
