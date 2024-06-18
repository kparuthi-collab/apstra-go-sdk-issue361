package apstra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	apiUrlFFRaGroups    = apiUrlBlueprintById + apiUrlPathDelim + "ra-groups"
	apiUrlFFRaGroupById = apiUrlFFRaGroups + apiUrlPathDelim + "%s"
)

var _ json.Unmarshaler = new(FreeformRaGroup)

type FreeformRaGroup struct {
	Id   ObjectId `json:"id,omitempty"`
	Data *FreeformRaGroupData
}

type FreeformRaGroupData struct {
	ParentId *ObjectId       `json:"parent_id"`
	Label    string          `json:"label"`
	Tags     []ObjectId      `json:"tags"`
	Data     json.RawMessage `json:"data"`
}

func (o *FreeformRaGroup) UnmarshalJSON(bytes []byte) error {
	var raw struct {
		Id       ObjectId        `json:"id"`
		ParentId *ObjectId       `json:"parent_id"`
		Label    string          `json:"label"`
		Tags     []ObjectId      `json:"tags"`
		Data     json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(bytes, &raw)
	if err != nil {
		return err
	}
	if o.Data == nil {
		o.Data = new(FreeformRaGroupData)
	}
	o.Id = raw.Id
	o.Data.ParentId = raw.ParentId
	o.Data.Label = raw.Label
	o.Data.Tags = raw.Tags
	o.Data.Data = raw.Data
	return err
}

func (o *FreeformClient) CreateRaGroup(ctx context.Context, in *FreeformRaGroupData) (ObjectId, error) {
	var response objectIdResponse
	err := o.client.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodPost,
		urlStr:      fmt.Sprintf(apiUrlFFRaGroups, o.blueprintId),
		apiInput:    in,
		apiResponse: &response,
	})
	if err != nil {
		return "", convertTtaeToAceWherePossible(err)
	}
	return response.Id, nil
}

func (o *FreeformClient) GetAllRaGroups(ctx context.Context) ([]FreeformRaGroup, error) {
	var response struct {
		Items []FreeformRaGroup `json:"items"`
	}
	err := o.client.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodGet,
		urlStr:      fmt.Sprintf(apiUrlFFRaGroups, o.blueprintId),
		apiResponse: &response,
	})
	if err != nil {
		return nil, convertTtaeToAceWherePossible(err)
	}
	return response.Items, nil
}

func (o *FreeformClient) GetRaGroup(ctx context.Context, id ObjectId) (*FreeformRaGroup, error) {
	response := new(FreeformRaGroup)
	err := o.client.talkToApstra(ctx, &talkToApstraIn{
		method:      http.MethodGet,
		urlStr:      fmt.Sprintf(apiUrlFFRaGroupById, o.blueprintId, id),
		apiResponse: response,
	})
	if err != nil {
		return nil, convertTtaeToAceWherePossible(err)
	}
	return response, nil
}

func (o *FreeformClient) UpdateRaGroup(ctx context.Context, id ObjectId, in *FreeformRaGroupData) error {
	err := o.client.talkToApstra(ctx, &talkToApstraIn{
		method:   http.MethodPatch,
		urlStr:   fmt.Sprintf(apiUrlFFRaGroupById, o.blueprintId, id),
		apiInput: in,
	})
	if err != nil {
		return convertTtaeToAceWherePossible(err)
	}
	return nil
}

func (o *FreeformClient) DeleteRaGroup(ctx context.Context, id ObjectId) error {
	err := o.client.talkToApstra(ctx, &talkToApstraIn{
		method: http.MethodDelete,
		urlStr: fmt.Sprintf(apiUrlFFRaGroupById, o.blueprintId, id),
	})
	if err != nil {
		return nil
	}
	return convertTtaeToAceWherePossible(err)
}
