package goapstra

import (
	"context"
	"log"
	"testing"
)

const (
	mixedTypeAnomaly = `{
      "actual": {
        "value": 1
      },
      "anomalous": {
        "value_min": 1,
		"value_max": "seven"
      },
      "anomaly_type": "probe",
      "id": "306b4f71-4285-4f1f-a106-e6571add182b",
      "identity": {
        "anomaly_type": "probe",
        "item_id": "b155214c-0378-41e7-8517-8fcc78ba00c9",
        "probe_id": "31fcc1ea-2538-492b-8f2f-1ef548c44e66",
        "probe_label": "VMs Without Fabric Configured VLANs",
        "properties": [
          {
            "key": "vlan",
            "value": 303
          },
          {
            "key": "hypervisor",
            "value": "1.2.3.4"
          },
          {
            "key": "interface",
            "value": "29acd19f-8cd0-4dd9-a60f-93eeaeff0595"
          },
          {
            "key": "virtual_machine_ip",
            "value": ""
          },
          {
            "key": "vnic",
            "value": "vm-166800:50:56:01:02:03"
          },
          {
            "key": "vnet",
            "value": "SomePortGroup"
          },
          {
            "key": "virtual_machine",
            "value": "testVmName"
          }
        ],
        "stage_name": "Affected VM Anomalies"
      },
      "last_modified_at": "2022-05-20T20:00:32.750806Z",
      "severity": "critical"
    }`

	singleTypeAnomaly = `{
      "actual": {
        "value": 1
      },
      "anomalous": {
        "value_min": 1
      },
      "anomaly_type": "probe",
      "id": "306b4f71-4285-4f1f-a106-e6571add182b",
      "identity": {
        "anomaly_type": "probe",
        "item_id": "b155214c-0378-41e7-8517-8fcc78ba00c9",
        "probe_id": "31fcc1ea-2538-492b-8f2f-1ef548c44e66",
        "probe_label": "VMs Without Fabric Configured VLANs",
        "properties": [
          {
            "key": "vlan",
            "value": 303
          },
          {
            "key": "hypervisor",
            "value": "1.2.3.4"
          },
          {
            "key": "interface",
            "value": "29acd19f-8cd0-4dd9-a60f-93eeaeff0595"
          },
          {
            "key": "virtual_machine_ip",
            "value": ""
          },
          {
            "key": "vnic",
            "value": "vm-166800:50:56:01:02:03"
          },
          {
            "key": "vnet",
            "value": "SomePortGroup"
          },
          {
            "key": "virtual_machine",
            "value": "testVmName"
          }
        ],
        "stage_name": "Affected VM Anomalies"
      },
      "last_modified_at": "2022-05-20T20:00:32.750806Z",
      "severity": "critical"
    }`
)

func (o *mockApstraApi) loadAnomalies() error {
	for _, s := range []string{singleTypeAnomaly, mixedTypeAnomaly} {
		o.anomalies = append(o.anomalies, s)
	}
	return nil
}

func TestUnpackAnomaly(t *testing.T) {
	var a *Anomaly
	a, err := unpackAnomaly([]byte(mixedTypeAnomaly))
	if err != nil {
		t.Fatal(err)
	}
	_ = a
}

func TestGetAnomalies(t *testing.T) {
	DebugLevel = 2
	clients, apis, err := getTestClientsAndMockAPIs()
	if err != nil {
		t.Fatal(err)
	}

	_ = apis
	_, mockExists := apis["mock"]
	if mockExists {
		err = apis["mock"].loadAnomalies()
		if err != nil {
			log.Fatal(err)
		}
	}

	for clientName, client := range clients {
		log.Printf("testing getAnomalies() with %s client", clientName)
		err := client.Login(context.TODO())
		if err != nil {
			t.Fatal(err)
		}

		anomalies, err := client.GetAnomalies(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("%d anomalies retrieved", len(anomalies))
	}
}
