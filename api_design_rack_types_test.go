package goapstra

import (
	"context"
	"log"
	"testing"
)

func TestListRackTypeIds(t *testing.T) {
	client, err := newLiveTestClient()
	if err != nil {
		t.Fatal(err)
	}

	rackTypeIds, err := client.listRackTypeIds(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	DebugLevel = 4

	rackTypes, err := client.getAllRackTypes(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	if len(rackTypeIds) != len(rackTypes) {
		t.Fatalf("got %d rack type IDs but %d rack types", len(rackTypeIds), len(rackTypes))
	}

}

func TestRackTypeStrings(t *testing.T) {
	type apiStringIota interface {
		String() string
		Int() int
	}

	type apiIotaString interface {
		parse() (int, error)
		string() string
	}

	type stringTestData struct {
		stringVal  string
		intType    apiStringIota
		stringType apiIotaString
	}
	testData := []stringTestData{
		{stringVal: "", intType: LeafRedundancyProtocolNone, stringType: leafRedundancyProtocolNone},
		{stringVal: "mlag", intType: LeafRedundancyProtocolMlag, stringType: leafRedundancyProtocolMlag},
		{stringVal: "esi", intType: LeafRedundancyProtocolEsi, stringType: leafRedundancyProtocolEsi},

		{stringVal: "", intType: AccessRedundancyProtocolNone, stringType: accessRedundancyProtocolNone},
		{stringVal: "esi", intType: AccessRedundancyProtocolEsi, stringType: accessRedundancyProtocolEsi},

		{stringVal: "l3clos", intType: FabricConnectivityDesignL3Clos, stringType: fabricConnectivityDesignL3Clos},
		{stringVal: "l3collapsed", intType: FabricConnectivityDesignL3Collapsed, stringType: fabricConnectivityDesignL3Collapsed},

		{stringVal: "singleAttached", intType: RackLinkAttachmentTypeSingle, stringType: rackLinkAttachmentTypeSingle},
		{stringVal: "dualAttached", intType: RackLinkAttachmentTypeDual, stringType: rackLinkAttachmentTypeDual},

		{stringVal: "", intType: RackLinkLagModeNone, stringType: rackLinkLagModeNone},
		{stringVal: "lacp_active", intType: RackLinkLagModeActive, stringType: rackLinkLagModeActive},
		{stringVal: "lacp_passive", intType: RackLinkLagModePassive, stringType: rackLinkLagModePassive},
		{stringVal: "static_lag", intType: RackLinkLagModeStatic, stringType: rackLinkLagModeStatic},
	}

	for i, td := range testData {
		ii := td.intType.Int()
		is := td.intType.String()
		sp, err := td.stringType.parse()
		if err != nil {
			t.Fatal(err)
		}
		ss := td.stringType.string()
		if td.intType.String() != td.stringType.string() ||
			td.intType.Int() != sp ||
			td.stringType.string() != td.stringVal {
			t.Fatalf("test index %d mismatch: %d %d '%s' '%s' '%s'",
				i, ii, sp, is, ss, td.stringVal)
		}
	}
}

func TestGetRackType(t *testing.T) {
	client, err := newLiveTestClient()
	if err != nil {
		t.Fatal(err)
	}

	rackTypes, err := client.listRackTypeIds(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	if len(rackTypes) == 0 {
		t.Skip("no rack types to fetch")
	}

	rt, err := client.getRackType(context.TODO(), rackTypes[0])
	if err != nil {
		t.Fatal(err)
	}

	log.Println(rt)
}

func TestCreateGetRackType(t *testing.T) {
	client, err := newLiveTestClient()
	if err != nil {
		t.Fatal(err)
	}

	leafLabel := "ll-" + randString(10, "hex")

	id, err := client.createRackType(context.TODO(), &RackType{
		DisplayName:              "rdn " + randString(5, "hex"),
		Description:              "description " + randString(10, "hex"),
		FabricConnectivityDesign: FabricConnectivityDesignL3Clos,
		LeafSwitches: []RackElementLeafSwitch{
			{
				Label:                       leafLabel,
				LeafLeafL3LinkCount:         0,
				LeafLeafL3LinkPortChannelId: 0,
				LeafLeafL3LinkSpeed:         nil,
				LeafLeafLinkCount:           0,
				LeafLeafLinkPortChannelId:   0,
				LeafLeafLinkSpeed:           nil,
				LinkPerSpineCount:           2,
				LinkPerSpineSpeed: &LogicalDevicePortSpeed{
					Unit:  "G",
					Value: 10,
				},
				MlagVlanId:         0,
				RedundancyProtocol: LeafRedundancyProtocolNone,
				Tags:               nil,
				LogicalDeviceId:    "virtual-7x10-1",
			},
		},
		GenericSystems: []RackElementGenericSystem{
			{
				Count:            5,
				AsnDomain:        FeatureSwitchEnabled,
				ManagementLevel:  GenericSystemUnmanaged,
				PortChannelIdMin: 0,
				PortChannelIdMax: 0,
				Loopback:         FeatureSwitchDisabled,
				Tags:             nil,
				Label:            "some generic system",
				Links: []RackLink{
					{
						Label:              "foo",
						Tags:               nil,
						LinkPerSwitchCount: 1,
						LinkSpeed: LogicalDevicePortSpeed{
							Unit:  "G",
							Value: 10,
						},
						TargetSwitchLabel: leafLabel,
						AttachmentType:    RackLinkAttachmentTypeSingle,
						LagMode:           RackLinkLagModeNone,
					},
				},
				LogicalDeviceId: "5ed7ed07-7222-4d6c-a5cb-1e1aa6036dab",
			},
		},
		AccessSwitches: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("id: '%s'\n", id)
}
