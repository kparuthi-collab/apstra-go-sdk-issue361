package aosSdk

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
)

func blueprintsTestClientCfg1() (*ClientCfg, error) {
	user, foundUser := os.LookupEnv(EnvApstraUser)
	pass, foundPass := os.LookupEnv(EnvApstraPass)
	scheme, foundScheme := os.LookupEnv(EnvApstraScheme)
	host, foundHost := os.LookupEnv(EnvApstraHost)
	portstr, foundPort := os.LookupEnv(EnvApstraPort)

	switch {
	case !foundUser:
		return nil, fmt.Errorf("environment variable '%s' not found", EnvApstraUser)
	case !foundPass:
		return nil, fmt.Errorf("environment variable '%s' not found", EnvApstraPass)
	case !foundScheme:
		return nil, fmt.Errorf("environment variable '%s' not found", EnvApstraScheme)
	case !foundHost:
		return nil, fmt.Errorf("environment variable '%s' not found", EnvApstraHost)
	case !foundPort:
		return nil, fmt.Errorf("environment variable '%s' not found", EnvApstraPort)
	}

	port, err := strconv.Atoi(portstr)
	if err != nil {
		return nil, fmt.Errorf("error converting '%s' to integer - %w", portstr, err)
	}

	return &ClientCfg{
		Scheme:    scheme,
		Host:      host,
		Port:      uint16(port),
		User:      user,
		Pass:      pass,
		TlsConfig: tls.Config{InsecureSkipVerify: true},
	}, nil
}

func TestGetBlueprints(t *testing.T) {
	clientCfg, err := blueprintsTestClientCfg1()
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(clientCfg)
	err = client.Login()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Logout()

	blueprints, err := client.GetBlueprints()
	if err != nil {
		t.Fatal(err)
	}

	result, err := json.Marshal(blueprints)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(string(result))
}