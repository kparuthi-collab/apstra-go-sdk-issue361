//go:build integration
// +build integration

package apstra

import (
	"context"
	"log"
	"testing"
)

func TestCreateUpdateDeleteRoutingZone(t *testing.T) {
	ctx := context.Background()

	clients, err := getTestClients(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	randStr := randString(5, "hex")
	label := "test-" + randStr
	vrfName := "test-" + randStr

	for clientName, client := range clients {
		bpClient, bpDel := testBlueprintA(ctx, t, client.client)
		defer func() {
			err := bpDel(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}()

		log.Printf("testing CreateSecurityZone() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		zoneId, err := bpClient.CreateSecurityZone(ctx, &SecurityZoneData{
			SzType:  SecurityZoneTypeEVPN,
			VrfName: vrfName,
			Label:   label,
		})
		if err != nil {
			t.Fatal(err)
		}

		log.Printf("created zone - id:'%s', name: '%s', label:'%s'", zoneId, vrfName, label)

		log.Println("fetching by id...")
		log.Printf("testing GetSecurityZone() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		zone, err := bpClient.GetSecurityZone(ctx, zoneId)
		if err != nil {
			t.Fatal(err)
		}
		if zone.Id != zoneId {
			t.Fatalf("created vs. fetched zone IDs don't match: '%s' and '%s'", zone.Id, zoneId)
		}

		log.Println("fetching by vrf name...")
		log.Printf("testing getSecurityZoneByVrfName() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		zone, err = bpClient.GetSecurityZoneByVrfName(ctx, vrfName)
		if err != nil {
			t.Fatal(err)
		}
		if zone.Id != zoneId {
			t.Fatalf("created vs. fetched zone IDs don't match: '%s' and '%s'", zone.Id, zoneId)
		}

		randStr2 := randString(5, "hex")
		vrfName2 := "test-" + randStr2
		label2 := "test-" + randStr2
		log.Printf("testing UpdateSecurityZone() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		err = bpClient.UpdateSecurityZone(ctx, zoneId, &SecurityZoneData{
			SzType:  SecurityZoneTypeEVPN,
			VrfName: vrfName2,
			Label:   label2,
		})
		if err != nil {
			t.Fatal(err)
		}

		log.Printf("testing GetSecurityZoneByVrfName() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		zone, err = bpClient.GetSecurityZoneByVrfName(ctx, vrfName2)
		if err != nil {
			t.Fatal(err)
		}
		if zone.Id != zoneId {
			t.Fatal()
		}
		if zone.Data.VrfName != vrfName2 {
			t.Fatal()
		}

		log.Printf("testing GetAllSecurityZones() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		zones, err := bpClient.GetAllSecurityZones(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if len(zones) != 2 {
			t.Fatalf("expected 2 security zones, got %d", len(zones))
		}

		ip4PoolIds, err := client.client.ListIp4PoolIds(ctx)
		if err != nil {
			t.Fatalf("error listing pool IDs - %s", err.Error())
		}

		ipv4PoolCount := len(ip4PoolIds)
		if ipv4PoolCount == 0 {
			t.Skip("an IPv4 pool is required for this test")
		}

		rga := &ResourceGroupAllocation{
			ResourceGroup: ResourceGroup{
				Type:           ResourceTypeIp4Pool,
				Name:           ResourceGroupNameLeafIp4,
				SecurityZoneId: &zoneId,
			},
			PoolIds: ip4PoolIds,
		}

		log.Printf("testing SetResourceAllocation() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		err = bpClient.SetResourceAllocation(ctx, rga)
		if err != nil {
			t.Fatal()
		}

		log.Printf("testing GetResourceAllocation() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		rga, err = bpClient.GetResourceAllocation(ctx, &rga.ResourceGroup)
		if err != nil {
			t.Fatal()
		}

		if ipv4PoolCount != len(rga.PoolIds) {
			t.Fatalf("expected %d pool IDs, got %d pool IDs", ipv4PoolCount, len(rga.PoolIds))
		}

		for i := 0; i < ipv4PoolCount; i++ {
			if ip4PoolIds[i] != rga.PoolIds[i] {
				t.Fatal("pool id mismatch")
			}
		}

		if *rga.ResourceGroup.SecurityZoneId != zoneId {
			t.Fatalf("expected security zone id %q, got %q", *rga.ResourceGroup.SecurityZoneId, zoneId)
		}

		rga.PoolIds = nil
		log.Printf("testing SetResourceAllocation() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		err = bpClient.SetResourceAllocation(ctx, rga)
		if err != nil {
			t.Fatal()
		}

		log.Printf("testing GetResourceAllocation() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		rga, err = bpClient.GetResourceAllocation(ctx, &rga.ResourceGroup)
		if err != nil {
			t.Fatal()
		}

		if len(rga.PoolIds) != 0 {
			t.Fatalf("expected 0 pool ids, got %d", len(rga.PoolIds))
		}

		log.Printf("testing DeleteSecurityZone() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		err = bpClient.DeleteSecurityZone(ctx, zoneId)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetDefaultRoutingZone(t *testing.T) {
	ctx := context.Background()

	clients, err := getTestClients(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	for clientName, client := range clients {
		bpClient, bpDel := testBlueprintA(ctx, t, client.client)
		defer func() {
			err := bpDel(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}()

		log.Printf("testing GetSecurityZoneByVrfName() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		sz, err := bpClient.GetSecurityZoneByVrfName(ctx, "default")
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("blueprint: %s - default security zone: %s", bpClient.blueprintId, sz.Id)
	}
}
