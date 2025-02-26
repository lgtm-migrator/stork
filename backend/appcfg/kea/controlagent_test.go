package keaconfig

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the Kea Control Agent configuration without comments is parsed.
func TestKeaControlAgentConfigurationFromJSON(t *testing.T) {
	// Arrange
	data := `{
		"Control-agent": {
			"http-host": "192.168.100.1",
			"http-port": 8001,
			"trust-anchor": "/home/user/stork-certs/CA",
			"cert-file": "/home/user/stork-certs/kea.crt",
			"key-file": "/home/user/stork-certs/kea.key",
			"cert-required": false
		}
	}`

	// Act
	config, err := NewFromJSON(data)

	// Assert
	require.NoError(t, err)
	require.True(t, config.IsControlAgent())
	host, ok := config.GetHTTPHost()
	require.True(t, ok)
	require.EqualValues(t, "192.168.100.1", host)
	port, ok := config.GetHTTPPort()
	require.True(t, ok)
	require.EqualValues(t, 8001, port)
	certFile, ok := config.GetCertFile()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/kea.crt", certFile)
	keyFile, ok := config.GetKeyFile()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/kea.key", keyFile)
	trustAnchor, ok := config.GetTrustAnchor()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/CA", trustAnchor)
	certRequired, ok := config.GetCertRequired()
	require.True(t, ok)
	require.False(t, certRequired)
}

// Test that the Kea Control Agent configuration with C style comments is parsed.
func TestKeaControlAgentConfigurationFromJSONWithCStyleComments(t *testing.T) {
	data := `{
		"Control-agent": { /*
			"http-host": "192.168.200.1",
			"http-port": 8002,
			"trust-anchor": "/home/user2/stork-certs/CA",
			"cert-file": "/home/user2/stork-certs/kea.crt",
			"key-file": "/home/user2/stork-certs/kea.key",
			"cert-required": false,
			*/
			"http-host": "192.168.100.1",
			//"http-port": 8003,
			"http-port": 8001, // "http-port": 8005,
			//"http-port": 8004,
			"trust-anchor": "/home/user/stork-certs/CA",
			"cert-file": "/home/user/stork-certs/kea.crt",
			"key-file": "/home/user/stork-certs/kea.key",
			"cert-required": false
		}
	}`

	// Act
	config, err := NewFromJSON(data)

	// Assert
	require.NoError(t, err)
	require.True(t, config.IsControlAgent())
	host, ok := config.GetHTTPHost()
	require.True(t, ok)
	require.EqualValues(t, "192.168.100.1", host)
	port, ok := config.GetHTTPPort()
	require.True(t, ok)
	require.EqualValues(t, 8001, port)
	certFile, ok := config.GetCertFile()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/kea.crt", certFile)
	keyFile, ok := config.GetKeyFile()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/kea.key", keyFile)
	trustAnchor, ok := config.GetTrustAnchor()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/CA", trustAnchor)
	certRequired, ok := config.GetCertRequired()
	require.True(t, ok)
	require.False(t, certRequired)
}

// Test that the Kea Control Agent configuration with hash comments is parsed.
func TestKeaControlAgentConfigurationFromJSONWithHashComments(t *testing.T) {
	// Arrange
	data := `{
		"Control-agent": {
			#"http-host": "192.168.100.2",
			#"http-port": 8001,
		#	"http-host": "192.168.100.3",
#			"http-port": 8003,
			"http-port": 8001, # "http-port": 8004,
			"http-host": "192.168.100.1",
			"trust-anchor": "/home/user/stork-certs/CA",
			"cert-file": "/home/user/stork-certs/kea.crt",
			#"cert-file": "/home/user2/stork-certs/kea.crt",
			"key-file": "/home/user/stork-certs/kea.key",
			"cert-required": false
		}
	}`

	// Act
	config, err := NewFromJSON(data)

	// Assert
	require.NoError(t, err)
	require.True(t, config.IsControlAgent())
	host, ok := config.GetHTTPHost()
	require.True(t, ok)
	require.EqualValues(t, "192.168.100.1", host)
	port, ok := config.GetHTTPPort()
	require.True(t, ok)
	require.EqualValues(t, 8001, port)
	certFile, ok := config.GetCertFile()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/kea.crt", certFile)
	keyFile, ok := config.GetKeyFile()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/kea.key", keyFile)
	trustAnchor, ok := config.GetTrustAnchor()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/CA", trustAnchor)
	certRequired, ok := config.GetCertRequired()
	require.True(t, ok)
	require.False(t, certRequired)
}

// Test that the Kea Control Agent configuration with minimal number of fields is parsed.
func TestKeaControlAgentConfigurationFromMinimalJSON(t *testing.T) {
	// Arrange
	data := `{
		"Control-agent": { }
	}`

	// Act
	config, err := NewFromJSON(data)

	// Assert
	require.NoError(t, err)
	require.True(t, config.IsControlAgent())
	host, ok := config.GetHTTPHost()
	require.False(t, ok)
	require.EqualValues(t, "127.0.0.1", host)
	port, ok := config.GetHTTPPort()
	require.False(t, ok)
	require.Zero(t, port)
	certFile, ok := config.GetCertFile()
	require.False(t, ok)
	require.Empty(t, certFile)
	keyFile, ok := config.GetKeyFile()
	require.False(t, ok)
	require.Empty(t, keyFile)
	trustAnchor, ok := config.GetTrustAnchor()
	require.False(t, ok)
	require.Empty(t, trustAnchor)
	_, ok = config.GetCertRequired()
	require.False(t, ok)
}

// Test that the empty string parsing returns an error.
func TestKeaControlAgentConfigurationFromEmptyString(t *testing.T) {
	// Arrange
	data := ""

	// Act
	config, err := NewFromJSON(data)

	// Assert
	require.Error(t, err)
	require.Nil(t, config)
}

// Test that the invalid JSON parsing returns an error.
func TestKeaControlAgentConfigurationFromInvalidJSON(t *testing.T) {
	// Arrange
	data := `{
		"Foo-Bar": {
			"http-port": 8001
		}
	}`

	// Act
	config, err := NewFromJSON(data)

	// Assert
	require.NoError(t, err)
	require.False(t, config.IsControlAgent())
	port, ok := config.GetHTTPPort()
	require.True(t, ok)
	require.EqualValues(t, 8001, port)
}

// Test that the real Kea Control Agent configuration is parsed.
func TestKeaControlAgentConfigurationFromFullJSON(t *testing.T) {
	// Arrange
	data := `
		// This is a basic configuration for the Kea Control Agent.
		//
		// Kea comes with a large suite of more than 30 configuration examples
		// and an extensive Kea Administrator Reference Manual (ARM). Please refer to
		// those materials to get a better understanding of what this software is able to
		// do. Comments in this configuration file sometimes indicate sections of
		// the Kea ARM where more details are available. The ARM comes with
		// each Kea download, but it is also available at
		// https://kea.readthedocs.io.
		//
		// This file contains only the Control Agent configuration.
		// The Control Agent ignores the configurations for any other Kea services that may
		// also be included in this file.
		{
		
		// RESTful interface to be available at http://127.0.0.1:8000/
		"Control-agent": {
			"authentication": {
			    "type": "basic",
			    "realm": "kea-control-agent",
			    "clients": [
			        {
			            "user": "foo",
			            "password": "bar"
			        }
			    ]
			},
			"http-host": "192.168.100.1",
			"http-port": 8001,
			"trust-anchor": "/home/user/stork-certs/CA",
			"cert-file": "/home/user/stork-certs/kea.crt",
			"key-file": "/home/user/stork-certs/kea.key",
			"cert-required": false,
		
			// Specify the location of the files to which the Control Agent
			// should connect to forward commands to the DHCPv4, DHCPv6,
			// and D2 servers via UNIX domain sockets.
			"control-sockets": {
				"dhcp4": {
					"socket-type": "unix",
					"socket-name": "/tmp/kea4-ctrl-socket"
				},
				"dhcp6": {
					"socket-type": "unix",
					"socket-name": "/tmp/kea6-ctrl-socket"
				},
				"d2": {
					"socket-type": "unix",
					"socket-name": "/tmp/kea-ddns-ctrl-socket"
				}
			},
		
			// Specify the hook libraries that are attached to the Control Agent.
			// Such hook libraries should support the 'control_command_receive'
			// hook point. This is currently commented out, since it has to
			// point to the existing hook library; otherwise the Control
			// Agent will not start.
			"hooks-libraries": [
		//  {
		//      "library": "/usr/lib/kea/hooks/control-agent-commands.so",
		//      "parameters": {
		//          "param1": "foo"
		//      }
		//  }
			],
		
		// Logging configuration starts here. Kea uses different loggers to log various
		// activities. For details (e.g. names of loggers), see
		// https://kea.readthedocs.io/en/latest/arm/logging.html.
			"loggers": [
			{
				// This specifies the logging for Control Agent daemon.
				"name": "kea-ctrl-agent",
				"output_options": [
					{
						// Specifies the output file. There are several special values
						// supported:
						// - stdout (prints on standard output)
						// - stderr (prints on standard error)
						// - syslog (logs to syslog)
						// - syslog:name (logs to syslog using specified name)
						// Any other value is considered a name of the file
						"output": "/var/log/kea-ctrl-agent.log"
		
						// This shorter log pattern is suitable for use with systemd,
						// and avoids redundant information.
						// "pattern": "%-5p %m\n"
		
						// This governs whether the log output is flushed to disk after
						// every write.
						// "flush": false,
		
						// This specifies the maximum size of the file before it is
						// rotated.
						// "maxsize": 1048576,
		
						// This specifies the maximum number of rotated files to keep.
						// "maxver": 8
					}
				],
				// This specifies the severity of log messages to keep. Supported values
				// are: FATAL, ERROR, WARN, INFO, DEBUG.
				"severity": "INFO",
		
				// If DEBUG level is specified, this value is used. 0 is least verbose,
				// 99 is most verbose. Be cautious: Kea can generate lots and lots
				// of logs if told to do so.
				"debuglevel": 0
			},
			{
			"name": "kea-ctrl-agent.auth",
			"severity": "DEBUG",
			"debuglevel": 99
			}
		]
		}
		}
	`

	// Act
	config, err := NewFromJSON(data)

	// Assert
	require.NoError(t, err)
	require.True(t, config.IsControlAgent())
	host, ok := config.GetHTTPHost()
	require.True(t, ok)
	require.EqualValues(t, "192.168.100.1", host)
	port, ok := config.GetHTTPPort()
	require.True(t, ok)
	require.EqualValues(t, 8001, port)
	certFile, ok := config.GetCertFile()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/kea.crt", certFile)
	keyFile, ok := config.GetKeyFile()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/kea.key", keyFile)
	trustAnchor, ok := config.GetTrustAnchor()
	require.True(t, ok)
	require.EqualValues(t, "/home/user/stork-certs/CA", trustAnchor)
	certRequired, ok := config.GetCertRequired()
	require.True(t, ok)
	require.False(t, certRequired)
}

// Test that the HTTP host is resolved to IP address.
func TestKeaControlAgentConfigurationResolveHost(t *testing.T) {
	// Arrange
	jsonNoHost := `{ "Control-agent": { } }`
	jsonEmptyHost := `{ "Control-agent": { "http-host": "" } }`
	jsonZeroHost := `{ "Control-agent": { "http-host": "0.0.0.0" } }`
	jsonColonHost := `{ "Control-agent": { "http-host": "::" } }`

	configNoHost, _ := NewFromJSON(jsonNoHost)
	configEmptyHost, _ := NewFromJSON(jsonEmptyHost)
	configZeroHost, _ := NewFromJSON(jsonZeroHost)
	configColonHost, _ := NewFromJSON(jsonColonHost)

	// Act
	hostNoHost, okNoHost := configNoHost.GetHTTPHost()
	hostEmptyHost, okEmptyHost := configEmptyHost.GetHTTPHost()
	hostZeroHost, okZeroHost := configZeroHost.GetHTTPHost()
	hostColonHost, okColonHost := configColonHost.GetHTTPHost()

	// Assert
	require.False(t, okNoHost)
	require.EqualValues(t, "127.0.0.1", hostNoHost)
	require.True(t, okEmptyHost)
	require.EqualValues(t, "127.0.0.1", hostEmptyHost)
	require.True(t, okZeroHost)
	require.EqualValues(t, "127.0.0.1", hostZeroHost)
	require.True(t, okColonHost)
	require.EqualValues(t, "::1", hostColonHost)
}

// Test that the secure protocol isn't detected when HTTPS configuration
// isn't complete.
func TestKeaControlAgentConfigurationDoNotUseSecureProtocol(t *testing.T) {
	// Arrange
	noSecure := []string{
		// Empty JSON
		`{ "Control-agent": { } }`,
		// Empty entries
		`{ "Control-agent": { "trust-anchor": "" } }`,
		`{ "Control-agent": { "cert-file": "" } }`,
		`{ "Control-agent": { "key-file": "" } }`,
		`{ "Control-agent": { "trust-anchor": "", "cert-file": "", "key-file": ""  } }`,
		// Filled single entries
		`{ "Control-agent": { "trust-anchor": "/path" } }`,
		`{ "Control-agent": { "cert-file": "/path" } }`,
		`{ "Control-agent": { "key-file": "/path" } }`,
		// Filled all entries except one
		`{ "Control-agent": { "trust-anchor": "/path", "cert-file": "/path", "key-file": ""  } }`,
	}

	for i, data := range noSecure {
		name := fmt.Sprintf("case-%d", i)
		testData := data // Fix 'scopelint' linter issue
		t.Run(name, func(t *testing.T) {
			config, _ := NewFromJSON(testData)
			// Act
			useSecure := config.UseSecureProtocol()

			// Assert
			require.False(t, useSecure)
		})
	}
}

// Test that the secure protocol is detected.
func TestKeaControlAgentConfigurationUseSecureProtocol(t *testing.T) {
	// Arrange
	data := `{
		"Control-agent": {
			"trust-anchor": "/foo",
			"cert-file": "/bar",
			"key-file": "/baz"
		}
	}`

	config, _ := NewFromJSON(data)

	// Act
	useSecure := config.UseSecureProtocol()

	// Assert
	require.True(t, useSecure)
}
