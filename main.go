// Usage: go run hotspot.go <ifname> [up|down|remove] [ssid] [password]

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	ourUUID = "7d706027-727c-4d4c-a816-f0e1b99db8ab"
)

var (
	defaultSSID     = "PIAP"
	defaultPassword = "raspberry"
)

func usage() {
	fmt.Printf("Usage: %s <ifname> [up|down|remove] [ssid] [password]\n", os.Args[0])
	os.Exit(0)
}

func startAccessPoint(conn *dbus.Conn, connPath, devPath dbus.ObjectPath) error {
	nmObj := conn.Object(
		"org.freedesktop.NetworkManager",
		"/org/freedesktop/NetworkManager",
	)

	// Activate the connection
	var activePath dbus.ObjectPath
	err := nmObj.
		Call("org.freedesktop.NetworkManager.ActivateConnection", 0,
			connPath, devPath, dbus.ObjectPath("/")).
		Store(&activePath)
	if err != nil {
		return fmt.Errorf("ActivateConnection failed: %v", err)
	}

	// Wait until the connection is activated
	props := conn.Object(
		"org.freedesktop.NetworkManager",
		activePath,
	)

	start := time.Now()
	for time.Since(start) < 10*time.Second {
		var stateVar dbus.Variant
		err = props.
			Call("org.freedesktop.DBus.Properties.Get", 0,
				"org.freedesktop.NetworkManager.Connection.Active",
				"State").
			Store(&stateVar)
		if err != nil {
			return fmt.Errorf("Properties.Get(State) failed: %v", err)
		}
		if state, ok := stateVar.Value().(uint32); ok && state == 2 {
			fmt.Println("Access point started")
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("failed to start access point")
}

func stopAccessPoint(conn *dbus.Conn, devPath dbus.ObjectPath) error {
	devObj := conn.Object(
		"org.freedesktop.NetworkManager",
		devPath,
	)
	err := devObj.
		Call("org.freedesktop.NetworkManager.Device.Disconnect", 0).
		Err
	if err != nil {
		return fmt.Errorf("Device.Disconnect failed: %v", err)
	}
	fmt.Println("Access point stopped")
	return nil
}

func removeConnection(conn *dbus.Conn, connPath dbus.ObjectPath) error {
	connObj := conn.Object(
		"org.freedesktop.NetworkManager",
		connPath,
	)
	err := connObj.
		Call("org.freedesktop.NetworkManager.Settings.Connection.Delete", 0).
		Err
	if err != nil {
		return fmt.Errorf("Connection.Delete failed: %v", err)
	}
	fmt.Println("Connection removed")
	return nil
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	iface := os.Args[1]
	op := os.Args[2]

	// Get SSID and password from args, env vars or use defaults
	ssid := defaultSSID
	password := defaultPassword

	// Check environment variables first
	if v, ok := os.LookupEnv("WIFI_SSID"); ok {
		ssid = v
	}
	if v, ok := os.LookupEnv("WIFI_PASSWORD"); ok {
		password = v
	}

	// Command line args override environment variables
	if op == "up" && len(os.Args) >= 5 {
		ssid = os.Args[3]
		password = os.Args[4]
	}

	// Connect to the system bus
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatalf("Failed to connect to system bus: %v", err)
	}

	// Get the Settings interface
	settingsObj := conn.Object(
		"org.freedesktop.NetworkManager",
		"/org/freedesktop/NetworkManager/Settings",
	)

	// List existing connections
	var paths []dbus.ObjectPath
	err = settingsObj.
		Call("org.freedesktop.NetworkManager.Settings.ListConnections", 0).
		Store(&paths)
	if err != nil {
		log.Fatalf("ListConnections failed: %v", err)
	}

	// Look up our connection by UUID
	var connPath dbus.ObjectPath
	for _, p := range paths {
		obj := conn.Object(
			"org.freedesktop.NetworkManager",
			p,
		)
		var cfg map[string]map[string]dbus.Variant
		err = obj.
			Call("org.freedesktop.NetworkManager.Settings.Connection.GetSettings", 0).
			Store(&cfg)
		if err != nil {
			continue
		}
		if v, ok := cfg["connection"]["uuid"].Value().(string); ok && v == ourUUID {
			connPath = p
			break
		}
	}

	// If not found, add it
	if connPath == "" {
		settingsMap := map[string]map[string]dbus.Variant{
			"connection": {
				"type":        dbus.MakeVariant("802-11-wireless"),
				"uuid":        dbus.MakeVariant(ourUUID),
				"id":          dbus.MakeVariant(ssid),
				"autoconnect": dbus.MakeVariant(false),
			},
			"802-11-wireless": {
				"ssid":    dbus.MakeVariant([]byte(ssid)),
				"mode":    dbus.MakeVariant("ap"),
				"band":    dbus.MakeVariant("bg"),
				"channel": dbus.MakeVariant(uint32(1)),
			},
			"802-11-wireless-security": {
				"key-mgmt": dbus.MakeVariant("wpa-psk"),
				"psk":      dbus.MakeVariant(password),
			},
			"ipv4": {
				"method": dbus.MakeVariant("shared"),
			},
			"ipv6": {
				"method": dbus.MakeVariant("ignore"),
			},
		}

		err = settingsObj.
			Call("org.freedesktop.NetworkManager.Settings.AddConnection", 0, settingsMap).
			Store(&connPath)
		if err != nil {
			log.Fatalf("AddConnection failed: %v", err)
		}
	}

	// Get the NetworkManager interface
	nmObj := conn.Object(
		"org.freedesktop.NetworkManager",
		"/org/freedesktop/NetworkManager",
	)

	// Find the device by interface name
	var devPath dbus.ObjectPath
	err = nmObj.
		Call("org.freedesktop.NetworkManager.GetDeviceByIpIface", 0, iface).
		Store(&devPath)
	if err != nil {
		log.Fatalf("GetDeviceByIpIface(%s) failed: %v", iface, err)
	}

	switch op {
	case "up":
		if err := startAccessPoint(conn, connPath, devPath); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)

	case "down":
		if err := stopAccessPoint(conn, devPath); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)

	case "remove":
		if connPath == "" {
			fmt.Println("No connection found to remove")
			os.Exit(0)
		}
		if err := removeConnection(conn, connPath); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)

	default:
		usage()
	}
}
