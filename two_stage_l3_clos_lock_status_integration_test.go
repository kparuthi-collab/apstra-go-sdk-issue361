//go:build integration
// +build integration

package goapstra

import (
	"context"
	"log"
	"testing"
)

func TestGetLockInfo(t *testing.T) {
	clients, err := getTestClients()
	if err != nil {
		t.Fatal(err)
	}

	for clientName, client := range clients {
		log.Printf("testing createBlueprintFromTemplate() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		name := randString(10, "hex")
		id, err := client.client.createBlueprintFromTemplate(context.TODO(), &CreateBlueprintFromTemplate{
			RefDesign:  RefDesignDatacenter,
			Label:      name,
			TemplateId: "L2_Virtual_EVPN",
		})
		if err != nil {
			t.Fatal(err)
		}

		bp, err := client.client.NewTwoStageL3ClosClient(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}

		l, err := bp.getLockInfo(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
		log.Println(l)
		log.Printf("got id '%s', deleting blueprint...\n", id)
		log.Printf("testing deleteBlueprint() against %s %s (%s)", client.clientType, clientName, client.client.ApiVersion())
		err = client.client.deleteBlueprint(context.TODO(), id)
		if err != nil {
			t.Fatal(err)
		}
	}
}