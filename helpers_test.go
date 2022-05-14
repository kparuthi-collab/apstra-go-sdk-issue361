package goapstra

import (
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func randString(n int, style string) string {
	rand.Seed(time.Now().UnixNano())

	var b64Letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-")
	var hexLetters = []rune("0123456789abcdef")
	var letters []rune
	b := make([]rune, n)
	switch style {
	case "hex":
		letters = hexLetters
	case "b64":
		letters = b64Letters
	}

	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randId() string {
	return randString(8, "hex") + "-" +
		randString(4, "hex") + "-" +
		randString(4, "hex") + "-" +
		randString(4, "hex") + "-" +
		randString(12, "hex")
}

func randJwt() string {
	return randString(36, "b64") + "." +
		randString(178, "b64") + "." +
		randString(86, "b64")
}

func TestKeyLogWriter(t *testing.T) {
	envVarName := randString(10, "hex")

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	testFileName := filepath.Join(dir, randString(10, "b64"))

	err = os.Setenv(envVarName, testFileName)
	if err != nil {
		t.Fatal(err)
	}

	klw, err := keyLogWriter(envVarName)
	if err != nil {
		t.Fatal(err)
	}

	data := randString(100, "b64")
	_, err = klw.Write([]byte(data))
	err = klw.Close()
	if err != nil {
		t.Fatal(err)
	}

	result, err := ioutil.ReadFile(testFileName)
	if err != nil {
		t.Fatal(err)
	}

	if string(result) != data {
		t.Fatal("data read and written do not match")
	}
}

func TestOurIpForPeer(t *testing.T) {
	test := net.ParseIP("127.0.0.1")
	expected := net.ParseIP("127.0.0.1")
	result, err := ourIpForPeer(test)
	if err != nil {
		t.Fatal(err)
	}
	switch {
	case test.String() == "<nil>":
		t.Fatal("test is '<nil>'")
	case expected.String() == "<nil>":
		t.Fatal("expected is '<nil>'")
	case result.String() == "<nil>":
		t.Fatal("result is '<nil>'")
	}
	if expected.String() != result.String() {
		t.Fatalf("expected %s, got %s", expected.String(), result.String())
	}
}
