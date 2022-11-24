/*
 * types.go
 * Copyright (C) 2022, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package dockercompose

type Container struct {
	Image       string
	Entrypoint  string
	Environment map[string]string
	Volumes     []string
	Ports       []string
	Networks    map[string]map[string]string
}
type NetworkConfig struct {
	Subnet  string
	Gateway string
}
type Network struct {
	Driver string
	Ipam   struct {
		Config []NetworkConfig
	}
}
type DockerComposeV3 struct {
	Version  string
	Services map[string]Container
	Networks map[string]Network
}
