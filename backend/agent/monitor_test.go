package agent

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetApps(t *testing.T) {
	am := NewAppMonitor()
	apps := am.GetApps()
	require.Len(t, apps, 0)
	am.Shutdown()
}

func TestGetCtrlAddressFromKeaConfigNonExisting(t *testing.T) {
	// check reading from non existing file
	path := "/tmp/non-exisiting-path"
	address, port := getCtrlAddressFromKeaConfig(path)
	require.Equal(t, int64(0), port)
	require.Empty(t, address)
}

func TestGetCtrlFromKeaConfigBadContent(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte("random content")
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// check reading from prepared file with bad content
	// so 0 should be returned as port
	address, port := getCtrlAddressFromKeaConfig(tmpFile.Name())
	require.Equal(t, int64(0), port)
	require.Empty(t, address)
}

func TestGetCtrlAddressFromKeaConfigOk(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte(string("\"http-host\": \"host.example.org\", \"http-port\": 1234"))
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// check reading from proper file
	address, port := getCtrlAddressFromKeaConfig(tmpFile.Name())
	require.Equal(t, int64(1234), port)
	require.Equal(t, "host.example.org", address)
}

func TestGetCtrlAddressFromKeaConfigAddress0000(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte(string("\"http-host\": \"0.0.0.0\", \"http-port\": 1234"))
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// check reading from proper file;
	// if CA is listening on 0.0.0.0 then 127.0.0.1 should be returned
	// as it is not possible to connect to 0.0.0.0
	address, port := getCtrlAddressFromKeaConfig(tmpFile.Name())
	require.Equal(t, int64(1234), port)
	require.Equal(t, "127.0.0.1", address)
}

func TestGetCtrlAddressFromKeaConfigAddressColons(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte(string("\"http-host\": \"::\", \"http-port\": 1234"))
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// check reading from proper file;
	// if CA is listening on :: then ::1 should be returned
	// as it is not possible to connect to ::
	address, port := getCtrlAddressFromKeaConfig(tmpFile.Name())
	require.Equal(t, int64(1234), port)
	require.Equal(t, "::1", address)
}

func TestDetectApps(t *testing.T) {
	am := NewAppMonitor()
	am.(*appMonitor).detectApps()
	am.Shutdown()
}

func TestDetectBind9App(t *testing.T) {
	// prepare named.conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte(string("keys \"foo\" {\n   algorithm \"hmac-md5\";\n   secret \"abcd\"; \n};\n"))
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	text = []byte(string("controls {\n   inet 127.0.0.53 port 5353 allow { localhost; } keys { \"foo\";};\n};"))
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// check BIND 9 app detection
	app := detectBind9App([]string{"", tmpFile.Name()})
	require.NotNil(t, app)
	require.Equal(t, app.Type, "bind9")
	require.Equal(t, 1, len(app.AccessPoints))
	ctrlPoint := app.AccessPoints[0]
	require.Equal(t, "control", ctrlPoint.Type)
	require.Equal(t, "127.0.0.53", ctrlPoint.Address)
	require.Equal(t, int64(5353), ctrlPoint.Port)
	require.Equal(t, "hmac-md5:abcd", ctrlPoint.Key)
}

func TestDetectKeaApp(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte("\"http-host\": \"localhost\", \"http-port\": 45634")
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	// check kea app detection
	app := detectKeaApp([]string{"", tmpFile.Name()})
	require.NotNil(t, app)
	require.Equal(t, "kea", app.Type)
	require.Equal(t, 1, len(app.AccessPoints))
	ctrlPoint := app.AccessPoints[0]
	require.Equal(t, "control", ctrlPoint.Type)
	require.Equal(t, "localhost", ctrlPoint.Address)
	require.Equal(t, int64(45634), ctrlPoint.Port)
	require.Empty(t, ctrlPoint.Key)
}
