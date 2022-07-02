package goapstra

import (
	"context"
	"log"
	"testing"
)

func TestListAndGetAllLogicalDevices(t *testing.T) {
	DebugLevel = 2
	clients, _, err := getTestClientsAndMockAPIs()
	if err != nil {
		t.Fatal(err)
	}
	log.Println(len(clients))

	for clientName, client := range clients {
		if clientName == "mock" {
			continue // todo have I given up on mock testing?
		}
		ids, err := client.listLogicalDeviceIds(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) <= 0 {
			t.Fatalf("only got %d ids from %s client", len(ids), clientName)
		}
		for _, id := range ids {
			ld, err := client.getLogicalDevice(context.TODO(), id)
			if err != nil {
				t.Fatal(err)
			}
			log.Printf("logical device id '%s' name '%s'\n", id, ld.DisplayName)
		}
	}
}
