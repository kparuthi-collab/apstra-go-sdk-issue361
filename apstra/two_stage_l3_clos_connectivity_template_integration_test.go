//go:build integration
// +build integration

package apstra

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"testing"
)

func compareCts(t *testing.T, a, b *ConnectivityTemplate, info string) {
	if *a.Id != *b.Id {
		t.Fatalf("%s CT IDs don't match: %q vs. %q", info, *a.Id, *b.Id)
	}

	if a.Label != b.Label {
		t.Fatalf("%s CT Labels don't match: %q vs. %q", info, a.Label, b.Label)
	}

	if a.Description != b.Description {
		t.Fatalf("%s CT Descriptoins don't match: %q vs. %q", info, a.Description, b.Description)
	}

	sort.Strings(a.Tags)
	sort.Strings(b.Tags)
	compareSlices(t, a.Tags, b.Tags, info)

	if len(a.Subpolicies) != len(b.Subpolicies) {
		t.Fatalf("%s CT primitive counts don't match: %d vs. %d", info, len(a.Subpolicies), len(b.Subpolicies))
	}

	aPrimitiveIds := make([]string, len(a.Subpolicies))
	aPrimitiveMap := make(map[ObjectId]*ConnectivityTemplatePrimitive, len(a.Subpolicies))
	for i, primitive := range a.Subpolicies {
		aPrimitiveIds[i] = primitive.Id.String()
		aPrimitiveMap[*primitive.Id] = primitive
	}

	bPrimitiveIds := make([]string, len(b.Subpolicies))
	bPrimitiveMap := make(map[ObjectId]*ConnectivityTemplatePrimitive, len(b.Subpolicies))
	for i, primitive := range b.Subpolicies {
		bPrimitiveIds[i] = primitive.Id.String()
		bPrimitiveMap[*primitive.Id] = primitive
	}

	sort.Strings(aPrimitiveIds)
	sort.Strings(bPrimitiveIds)
	compareSlices(t, aPrimitiveIds, bPrimitiveIds, info)

	for k := range aPrimitiveMap {
		compareCtPrimitives(t, aPrimitiveMap[k], bPrimitiveMap[k], info)
	}
}

func compareCtPrimitives(t *testing.T, a, b *ConnectivityTemplatePrimitive, info string) {
	if *a.Id != *b.Id {
		t.Fatalf("%s CT Primitive IDs don't match: %q vs. %q", info, *a.Id, *b.Id)
	}

	if a.BatchId == nil && b.BatchId != nil {
		t.Fatalf("%s Primitive 'a' batch ID is nil, but 'b's is %q", info, b.BatchId)
	}

	if a.BatchId != nil && b.BatchId == nil {
		t.Fatalf("%s Primitive 'a' batch ID is %q, but 'b's is nil", info, a.BatchId)
	}

	if a.BatchId != nil && b.BatchId != nil && *a.BatchId != *b.BatchId {
		t.Fatalf("%s CT Primitive IDs don't match: %q vs. %q", info, *a.BatchId, *b.BatchId)
	}

	if *a.PipelineId != *b.PipelineId {
		t.Fatalf("%s CT Primitive IDs don't match: %q vs. %q", info, *a.PipelineId, *b.PipelineId)
	}

	aRawAttributes, err := a.Attributes.raw()
	if err != nil {
		t.Fatal(err)
	}
	bRawAttributes, err := b.Attributes.raw()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(aRawAttributes, bRawAttributes) != 0 {
		t.Fatalf("%s CT primitive attributes are not equal: %q vs. %q", info, string(aRawAttributes), string(bRawAttributes))
	}

	aPrimitiveIds := make([]string, len(a.Subpolicies))
	aPrimitiveMap := make(map[ObjectId]*ConnectivityTemplatePrimitive, len(a.Subpolicies))
	for i, primitive := range a.Subpolicies {
		aPrimitiveIds[i] = primitive.Id.String()
		aPrimitiveMap[*primitive.Id] = primitive
	}

	bPrimitiveIds := make([]string, len(b.Subpolicies))
	bPrimitiveMap := make(map[ObjectId]*ConnectivityTemplatePrimitive, len(b.Subpolicies))
	for i, primitive := range b.Subpolicies {
		bPrimitiveIds[i] = primitive.Id.String()
		bPrimitiveMap[*primitive.Id] = primitive
	}

	sort.Strings(aPrimitiveIds)
	sort.Strings(bPrimitiveIds)
	compareSlices(t, aPrimitiveIds, bPrimitiveIds, info)

	for k := range aPrimitiveMap {
		compareCtPrimitives(t, aPrimitiveMap[k], bPrimitiveMap[k], info)
	}
}

func TestCreateGetUpdateDeleteCT(t *testing.T) {
	ctx := context.Background()

	clients, err := getTestClients(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	randomTags := make([]string, rand.Intn(5)+2)
	for i := range randomTags {
		randomTags[i] = randString(5, "hex")
	}

	type testCase struct {
		ct      *ConnectivityTemplate
		eStatus CtPrimitiveStatus
	}

	update := &ConnectivityTemplate{
		Label:       randString(5, "hex"),
		Description: randString(5, "hex"),
		Subpolicies: []*ConnectivityTemplatePrimitive{
			{
				Attributes: &ConnectivityTemplatePrimitiveAttributesAttachSingleVlan{},
			},
		},
		Tags: []string{randString(5, "hex")},
	}

	for clientName, client := range clients {
		bpClient, bpDel := testBlueprintA(ctx, t, client.client)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err := bpDel(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}()

		sz, err := bpClient.GetSecurityZoneByVrfName(ctx, "default")
		if err != nil {
			t.Fatal(err)
		}

		vlan10 := Vlan(10)

		testCases := []testCase{
			{
				ct: &ConnectivityTemplate{
					Subpolicies: nil,
					Tags:        nil,
				},
				eStatus: CtPrimitiveStatusIncomplete,
			},
			{
				ct: &ConnectivityTemplate{
					Label:       randString(5, "hex"),
					Description: randString(10, "hex"),
					Subpolicies: nil,
					Tags:        randomTags,
				},
				eStatus: CtPrimitiveStatusIncomplete,
			},
			{
				ct: &ConnectivityTemplate{
					Label:       randString(5, "hex"),
					Description: randString(10, "hex"),
					Subpolicies: []*ConnectivityTemplatePrimitive{
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachLogicalLink{
								SecurityZone:       &sz.Id,
								Tagged:             false,
								IPv4AddressingType: CtPrimitiveIPv4AddressingTypeNumbered,
								IPv6AddressingType: CtPrimitiveIPv6AddressingTypeLinkLocal,
							},
						},
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachLogicalLink{
								SecurityZone:       &sz.Id,
								Tagged:             true,
								Vlan:               &vlan10,
								IPv4AddressingType: CtPrimitiveIPv4AddressingTypeNumbered,
								IPv6AddressingType: CtPrimitiveIPv6AddressingTypeLinkLocal,
							},
							Subpolicies: []*ConnectivityTemplatePrimitive{
								{
									Attributes: &ConnectivityTemplatePrimitiveAttributesAttachStaticRoute{
										ShareIpEndpoint: true,
										Network: &net.IPNet{
											IP:   net.IP{1, 1, 1, 1},
											Mask: net.IPMask{255, 255, 255, 255},
										},
									},
								},
								{
									Attributes: &ConnectivityTemplatePrimitiveAttributesAttachStaticRoute{
										ShareIpEndpoint: true,
										Network: &net.IPNet{
											IP:   net.IP{2, 2, 2, 2},
											Mask: net.IPMask{255, 255, 255, 255},
										},
									},
								},
							},
						},
					},
					Tags: randomTags,
				},
				eStatus: CtPrimitiveStatusReady,
			},
		}

		for i, tc := range testCases {
			err = tc.ct.SetIds()
			if err != nil {
				t.Fatal(err)
			}

			err = tc.ct.SetUserData()
			if err != nil {
				t.Fatal(err)
			}

			log.Printf("testing CreateConnectivityTemplate(%d) against %s %s (%s)", i, client.clientType, clientName, client.client.ApiVersion())
			err = bpClient.CreateConnectivityTemplate(ctx, tc.ct)
			if err != nil {
				t.Fatal(err)
			}

			log.Printf("testing GetConnectivityTemplate(%d) against %s %s (%s)", i, client.clientType, clientName, client.client.ApiVersion())
			read, err := bpClient.GetConnectivityTemplate(ctx, *tc.ct.Id)
			if err != nil {
				t.Fatal(err)
			}
			compareCts(t, tc.ct, read, fmt.Sprintf("while comparing connectivity templates test case %d:", i))

			log.Printf("testing GetConnectivityTemplateState(%d) against %s %s (%s)", i, client.clientType, clientName, client.client.ApiVersion())
			ctState, err := bpClient.GetConnectivityTemplateState(ctx, *tc.ct.Id)
			if err != nil {
				t.Fatal(err)
			}
			if tc.eStatus != ctState.Status {
				t.Fatalf("expected status %q, got status %q", tc.eStatus.String(), ctState.Status.String())
			}

			u := update
			id := *tc.ct.Id
			u.Id = &id
			err = u.SetIds()
			if err != nil {
				t.Fatal(err)
			}

			err = u.SetUserData()
			if err != nil {
				t.Fatal(err)
			}

			log.Printf("testing UpdateConnectivityTemplate(%d) against %s %s (%s)", i, client.clientType, clientName, client.client.ApiVersion())
			err = bpClient.UpdateConnectivityTemplate(ctx, u)
			if err != nil {
				t.Fatal(err)
			}

			log.Printf("testing GetConnectivityTemplate(%d) against %s %s (%s)", i, client.clientType, clientName, client.client.ApiVersion())
			read, err = bpClient.GetConnectivityTemplate(ctx, *tc.ct.Id)
			if err != nil {
				t.Fatal(err)
			}
			compareCts(t, u, read, fmt.Sprintf("while comparing connectivity templates test case %d:", i))

			log.Printf("testing DeleteConnectivityTemplate(%d) against %s %s (%s)", i, client.clientType, clientName, client.client.ApiVersion())
			err = bpClient.DeleteConnectivityTemplate(ctx, *tc.ct.Id)
			if err != nil {
				t.Fatal(err)
			}

			log.Printf("testing GetConnectivityTemplate(%d) against %s %s (%s)", i, client.clientType, clientName, client.client.ApiVersion())
			read, err = bpClient.GetConnectivityTemplate(ctx, *tc.ct.Id)
			if err == nil {
				t.Fatal("GetConnectivityTemplate() against deleted ID should have produced an error")
			}
			var ace ClientErr
			if !(errors.As(err, &ace) && ace.errType == ErrNotfound) {
				t.Fatalf("expected ErrNotFound, got %s", err.Error())
			}
		}
	}
}

func TestCtLayout(t *testing.T) {
	ctx := context.Background()

	clients, err := getTestClients(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	for _, client := range clients {
		bpClient, bpDel := testBlueprintA(ctx, t, client.client)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err := bpDel(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}()

		err = bpClient.SetFabricAddressingPolicy(ctx, &TwoStageL3ClosFabricAddressingPolicy{Ipv6Enabled: true})
		if err != nil {
			t.Fatal(err)
		}

		sz, err := bpClient.GetSecurityZoneByVrfName(ctx, "default")
		if err != nil {
			t.Fatal(err)
		}

		rp, err := bpClient.GetRoutingPolicyByName(ctx, "Default_immutable")
		if err != nil {
			t.Fatal(err)
		}

		//sz := SecurityZone{}
		//rp := DcRoutingPolicy{}
		_ = client

		vlan := Vlan(11)
		ct := ConnectivityTemplate{
			Label: randString(5, "hex"),
			Subpolicies: []*ConnectivityTemplatePrimitive{
				{
					Attributes: &ConnectivityTemplatePrimitiveAttributesAttachLogicalLink{
						//SecurityZone:       &sz.Id,
						SecurityZone:       &sz.Id,
						Tagged:             false,
						Vlan:               &vlan,
						IPv4AddressingType: CtPrimitiveIPv4AddressingTypeNumbered,
						IPv6AddressingType: CtPrimitiveIPv6AddressingTypeNumbered,
					},
					Subpolicies: []*ConnectivityTemplatePrimitive{
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachBgpWithPrefixPeeringForSviOrSubinterface{
								Ipv4Safi:              true,
								SessionAddressingIpv4: true,
							},
							Subpolicies: []*ConnectivityTemplatePrimitive{
								{Attributes: &ConnectivityTemplatePrimitiveAttributesAttachExistingRoutingPolicy{RpToAttach: &rp.Id}},
								{Attributes: &ConnectivityTemplatePrimitiveAttributesAttachExistingRoutingPolicy{RpToAttach: &rp.Id}},
							},
						},
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachBgpWithPrefixPeeringForSviOrSubinterface{
								Ipv6Safi:              true,
								SessionAddressingIpv6: true,
							},
							Subpolicies: []*ConnectivityTemplatePrimitive{
								{Attributes: &ConnectivityTemplatePrimitiveAttributesAttachExistingRoutingPolicy{RpToAttach: &rp.Id}},
								{Attributes: &ConnectivityTemplatePrimitiveAttributesAttachExistingRoutingPolicy{RpToAttach: &rp.Id}},
							},
						},
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachStaticRoute{
								Network: &net.IPNet{
									IP:   []byte{1, 1, 1, 1},
									Mask: []byte{255, 255, 255, 255},
								},
							},
						},
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachStaticRoute{
								Network: &net.IPNet{
									IP:   []byte{2, 2, 2, 2},
									Mask: []byte{255, 255, 255, 255},
								},
							},
						},
					},
				},
				{
					Attributes: &ConnectivityTemplatePrimitiveAttributesAttachLogicalLink{
						SecurityZone:       &sz.Id,
						Tagged:             false,
						Vlan:               &vlan,
						IPv4AddressingType: CtPrimitiveIPv4AddressingTypeNumbered,
						IPv6AddressingType: CtPrimitiveIPv6AddressingTypeNumbered,
					},
					Subpolicies: []*ConnectivityTemplatePrimitive{
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachBgpWithPrefixPeeringForSviOrSubinterface{
								Ipv4Safi:              true,
								SessionAddressingIpv4: true,
							},
							Subpolicies: []*ConnectivityTemplatePrimitive{
								{Attributes: &ConnectivityTemplatePrimitiveAttributesAttachExistingRoutingPolicy{RpToAttach: &rp.Id}},
								{Attributes: &ConnectivityTemplatePrimitiveAttributesAttachExistingRoutingPolicy{RpToAttach: &rp.Id}},
							},
						},
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachBgpWithPrefixPeeringForSviOrSubinterface{
								Ipv6Safi:              true,
								SessionAddressingIpv6: true,
							},
							Subpolicies: []*ConnectivityTemplatePrimitive{
								{Attributes: &ConnectivityTemplatePrimitiveAttributesAttachExistingRoutingPolicy{RpToAttach: &rp.Id}},
								{Attributes: &ConnectivityTemplatePrimitiveAttributesAttachExistingRoutingPolicy{RpToAttach: &rp.Id}},
							},
						},
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachStaticRoute{
								Network: &net.IPNet{
									IP:   []byte{3, 3, 3, 3},
									Mask: []byte{255, 255, 255, 255},
								},
							},
						},
						{
							Attributes: &ConnectivityTemplatePrimitiveAttributesAttachStaticRoute{
								Network: &net.IPNet{
									IP:   []byte{4, 4, 4, 4},
									Mask: []byte{255, 255, 255, 255},
								},
							},
						},
					},
				},
			},
		}
		err = ct.SetIds()
		if err != nil {
			t.Fatal(err)
		}

		err = ct.SetUserData()
		if err != nil {
			t.Fatal(err)
		}

		err = bpClient.CreateConnectivityTemplate(ctx, &ct)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestConnectivityTemplate404(t *testing.T) {
	ctx := context.Background()

	clients, err := getTestClients(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	for _, client := range clients {
		bpClient, bpDel := testBlueprintA(ctx, t, client.client)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err := bpDel(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}()

		_, err = bpClient.GetConnectivityTemplate(ctx, "bogus")
		if err == nil {
			t.Fatal("retrieval of bogus CT should have produced an error")
		} else {
			var ace ClientErr
			if !errors.As(err, &ace) || ace.Type() != ErrNotfound {
				t.Fatal("error should have been something 404-ish")
			}
		}

		err = bpClient.DeleteConnectivityTemplate(ctx, "bogus")
		if err == nil {
			t.Fatal("deletion of bogus CT should have produced an error")
		} else {
			var ace ClientErr
			if !errors.As(err, &ace) || ace.Type() != ErrNotfound {
				t.Fatal("error should have been something 404-ish")
			}
		}
	}
}
