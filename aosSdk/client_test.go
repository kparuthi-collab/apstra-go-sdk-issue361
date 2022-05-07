package aosSdk

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"testing"
)

func clientTestClient() (*Client, error) {
	user, foundUser := os.LookupEnv(EnvApstraUser)
	pass, foundPass := os.LookupEnv(EnvApstraPass)
	scheme, foundScheme := os.LookupEnv(EnvApstraScheme)
	host, foundHost := os.LookupEnv(EnvApstraHost)
	portStr, foundPort := os.LookupEnv(EnvApstraPort)
	keyLogFile, foundkeyLogFile := os.LookupEnv(EnvApstraApiKeyLogFile)

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

	var kl io.Writer
	var err error
	if foundkeyLogFile {
		kl, err = keyLogWriter(keyLogFile)
		if err != nil {
			return nil, fmt.Errorf("error creating keyLogWriter - %w", err)
		}
	} else {
		kl = nil
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("error converting '%s' to integer - %w", portStr, err)
	}

	client := NewClient(&ClientCfg{
		Scheme:    scheme,
		Host:      host,
		Port:      uint16(port),
		User:      user,
		Pass:      pass,
		TlsConfig: tls.Config{InsecureSkipVerify: true, KeyLogWriter: kl},
	})

	return client, nil
}

func TestNewClient(t *testing.T) {
	client, err := clientTestClient()
	if err != nil {
		t.Fatal(err)
	}

	log.Println(client.baseUrl)
}

func TestParseBytesAsTaskId(t *testing.T) {
	var testData [][]byte
	var expected []bool

	testData = append(testData, []byte(""))
	expected = append(expected, false)

	testData = append(testData, []byte("{}"))
	expected = append(expected, false)

	testData = append(testData, []byte("[]"))
	expected = append(expected, false)

	if len(testData) != len(expected) {
		t.Fatalf("test setup error - have %d tests, but expect %d results", len(testData), len(expected))
	}

	for i, td := range testData {
		result := &taskIdResponse{}
		ok := parseBytesAsTaskId(td, result)
		if ok != expected[i] {
			t.Fatalf("test data '%s' produced '%t', expected '%t'", string(td), ok, expected[i])
		}
	}
}