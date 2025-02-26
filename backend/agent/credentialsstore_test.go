package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the credentials store is constructed correctly.
func TestCreateStore(t *testing.T) {
	store := NewCredentialsStore()
	require.NotNil(t, store)
	require.Len(t, store.basicAuthCredentials, 0)
}

// Test that the Basic Auth credentials are constructed correctly.
func TestCreateBasicAuthCredentials(t *testing.T) {
	credentials := NewBasicAuthCredentials("foo", "bar")
	require.NotNil(t, credentials)
	require.EqualValues(t, "foo", credentials.User)
	require.EqualValues(t, "bar", credentials.Password)
}

// Test that the Basic Auth credentials are added to store correctly.
func TestAddBasicAuthCredentials(t *testing.T) {
	store := NewCredentialsStore()
	credentials := NewBasicAuthCredentials("foo", "bar")
	err := store.AddOrUpdateBasicAuth("127.0.0.1", 1, credentials)
	require.NoError(t, err)
	fetchedCredentials, ok := store.GetBasicAuth("127.0.0.1", 1)
	require.True(t, ok)
	require.NotNil(t, fetchedCredentials)
	require.EqualValues(t, "foo", fetchedCredentials.User)
	require.EqualValues(t, "bar", fetchedCredentials.Password)
}

// Test that the store accepts only valid IP addresses.
func TestAddBasicAuthCredentialsInvalidIPs(t *testing.T) {
	ipAddresses := []string{
		"",
		"foo",
		"ZZ:ZZ::",
		"0",
		":",
		".",
		"19216801",
		"192..168.0.1",
		"FF:::FF:FF::",
		"FF:FF:FFFFFF::",
		"-192.168.0.1",
	}

	store := NewCredentialsStore()
	credentials := NewBasicAuthCredentials("foo", "bar")

	for _, ip := range ipAddresses {
		err := store.AddOrUpdateBasicAuth(ip, 1, credentials)
		require.Error(t, err, "IP: %s", ip)
	}
}

// Test that the empty Basic Auth credentials (without user and password)
// are added to store correctly.
func TestAddBasicAuthEmptyCredentials(t *testing.T) {
	store := NewCredentialsStore()
	credentials := NewBasicAuthCredentials("", "")
	err := store.AddOrUpdateBasicAuth("127.0.0.1", 1, credentials)
	require.NoError(t, err)
	fetchedCredentials, ok := store.GetBasicAuth("127.0.0.1", 1)
	require.True(t, ok)
	require.NotNil(t, fetchedCredentials)
	require.Empty(t, fetchedCredentials.User)
	require.Empty(t, fetchedCredentials.Password)
}

// Test that the Basic Auth credentials are updated correctly.
func TestUpdateBasicAuthCredentials(t *testing.T) {
	store := NewCredentialsStore()
	credentials := NewBasicAuthCredentials("foo", "bar")
	err := store.AddOrUpdateBasicAuth("127.0.0.1", 1, credentials)
	require.NoError(t, err)
	credentials = NewBasicAuthCredentials("oof", "rab")
	err = store.AddOrUpdateBasicAuth("127.0.0.1", 1, credentials)
	require.NoError(t, err)
	fetchedCredentials, ok := store.GetBasicAuth("127.0.0.1", 1)
	require.True(t, ok)
	require.NotNil(t, fetchedCredentials)
	require.EqualValues(t, "oof", fetchedCredentials.User)
	require.EqualValues(t, "rab", fetchedCredentials.Password)
}

// Test that the Basic Auth credentials are deleted correctly.
func TestDeleteBasicAuthCredentials(t *testing.T) {
	store := NewCredentialsStore()
	credentials := NewBasicAuthCredentials("foo", "bar")
	err := store.AddOrUpdateBasicAuth("127.0.0.1", 1, credentials)
	require.NoError(t, err)
	store.RemoveBasicAuth("127.0.0.1", 1)
	fetchedCredentials, ok := store.GetBasicAuth("127.0.0.1", 1)
	require.False(t, ok)
	require.Nil(t, fetchedCredentials)
}

// Test fetching non-existing Basic Auth credentials. It should
// return nil and proper (falsy) status.
func TestGetMissingBasicAuthCredentials(t *testing.T) {
	store := NewCredentialsStore()
	fetchedCredentials, ok := store.GetBasicAuth("127.0.0.1", 1)
	require.False(t, ok)
	require.Nil(t, fetchedCredentials)
}

// Get the Basic Auth credentials by URL.
func TestGetBasicAuthCredentialsByURL(t *testing.T) {
	store := NewCredentialsStore()
	credentials := NewBasicAuthCredentials("foo", "bar")
	err := store.AddOrUpdateBasicAuth("127.0.0.1", 1, credentials)
	require.NoError(t, err)

	validURLs := []string{
		"http://127.0.0.1:1",
		"https://127.0.0.1:1",
		"http://127.0.0.1:1/",
		"http://127.0.0.1:1?query=param",
		"http://127.0.0.1:1/segment",
	}
	invalidURLs := []string{
		"http://baz:1",
		"http://foo:1",
		"http://127.0.0.1:2",
		"http://:1",
		"http://127.0.0.1",
		"",
		"127.0.0.1",
		"1",
		"protocol://127.0.0.1:1",
		"127.0.0.1:1",
	}

	for _, url := range validURLs {
		fetchedCredentials, ok := store.GetBasicAuthByURL(url)
		require.True(t, ok, "URL: %s", url)
		require.NotNil(t, fetchedCredentials)
		require.EqualValues(t, "foo", fetchedCredentials.User)
		require.EqualValues(t, "bar", fetchedCredentials.Password)
	}

	for _, url := range invalidURLs {
		fetchedCredentials, ok := store.GetBasicAuthByURL(url)
		require.False(t, ok)
		require.Nil(t, fetchedCredentials)
	}
}

// Test read the store from the proper JSON content.
func TestReadStoreFromProperContent(t *testing.T) {
	store := NewCredentialsStore()
	content := strings.NewReader(`{
		"basic_auth": [
			{
				"ip": "192.168.0.1",
				"port": 1234,
				"user": "foo",
				"password": "bar"
			}
		]
	}`)

	err := store.Read(content)
	require.NoError(t, err)
	credentials, ok := store.GetBasicAuth("192.168.0.1", 1234)
	require.True(t, ok)
	require.NotNil(t, credentials)
	require.EqualValues(t, "foo", credentials.User)
	require.EqualValues(t, "bar", credentials.Password)
}

// IP addresses may be written by humans in some different forms.
// They may be defined using any or mixed letter case.
// The credentials store should normalize all differences.
func TestReadStoreFromFileWithAbbreviations(t *testing.T) {
	store := NewCredentialsStore()
	content := strings.NewReader(`{
		"basic_auth": [
			{
				"ip": "127.0.0.1",
				"port": 1,
				"user": "a",
				"password": "aa"
			},
			{
				"ip": "::1",
				"port": 2,
				"user": "b",
				"password": "bb"
			},
			{
				"ip": "2001:db8:0000::",
				"port": 3,
				"user": "c",
				"password": "cc"
			},
			{
				"ip": "::1234:5678:91.123.4.56",
				"port": 4,
				"user": "d",
				"password": "dd"
			},
			{
				"ip": "2001:0000:0000:0000:0000:0000:0000:FFFF",
				"port": 5,
				"user": "e",
				"password": "ee"
			}
		]
	}`)

	err := store.Read(content)
	require.NoError(t, err)

	addresses := []string{
		"127.0.0.1",
		"::1",
		"2001:db8::",
		"::1234:5678:5b7b:438",
		"2001::ffff",
	}

	for idx, address := range addresses {
		port := idx + 1
		expectedUser := string(rune('a' + idx))
		expectedPassword := expectedUser + expectedUser
		credentials, ok := store.GetBasicAuth(address, int64(port))
		require.True(t, ok, "Address: %s", address)
		require.NotNil(t, credentials)
		require.EqualValues(t, expectedUser, credentials.User)
		require.EqualValues(t, expectedPassword, credentials.Password)
	}
}

// Test abbreviation normalization.
func TestAbbreviationNormalization(t *testing.T) {
	store := NewCredentialsStore()
	credentials := NewBasicAuthCredentials("foo", "bar")
	err := store.AddOrUpdateBasicAuth("FF:FF:0000:0000::", 42, credentials)
	require.NoError(t, err)
	credentials2, ok := store.GetBasicAuth("FF:FF::", 42)
	require.True(t, ok)
	require.EqualValues(t, credentials, credentials2)
	store.RemoveBasicAuth("FF:FF:0000::", 42)
	credentials3, ok := store.GetBasicAuth("FF:FF::", 42)
	require.False(t, ok)
	require.Nil(t, credentials3)
}

// Test read the store from the invalid JSON content.
func TestReadStoreFromInvalidContent(t *testing.T) {
	store := NewCredentialsStore()

	type unitData struct {
		Name    string
		Content string
	}

	items := []unitData{
		{
			"Empty content", ``,
		},
		{
			"Port is not a number",
			`{
				"basic_auth": [
					{
						"ip": "192.168.0.1",
						"port": "1234",
						"user": "foo",
						"password": "bar"
					}
				]
			}`,
		},
		{
			"Missing port",
			`{
				"basic_auth": [
					{
						"ip": "192.168.0.1",
						"user": "foo",
						"password": "bar"
					}
				]
			}`,
		},
		{
			"Missing all fields",
			`{
				"basic_auth": [
					{ }
				]
			}`,
		},
		{
			"Missing key quotes",
			`{
				basic_auth: [
					{
						ip: "192.168.0.1",
						port: 8000
						user: "foo",
						password: "bar"
					}
				]
			}`,
		},
	}

	for _, item := range items {
		testContent := item.Content
		t.Run(item.Name, func(t *testing.T) {
			reader := strings.NewReader(testContent)
			err := store.Read(reader)
			require.Error(t, err, "Content:", testContent)
			require.Len(t, store.basicAuthCredentials, 0)
		})
	}
}
