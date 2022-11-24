/*
 * docker_compose.go
 * Copyright (C) 2022, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package dockercompose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"chain4travel.com/camktncr/pkg/version1"
	"github.com/ava-labs/avalanchego/genesis"
	"github.com/ava-labs/avalanchego/utils/perms"
	"gopkg.in/yaml.v3"
)

var (
	BASE_DIR        = "./local/docker-compose"
	NETWORK_ADDRESS = "10.0.7."
)

func CreateKeyAndCertFile(stakers []version1.Staker, genesisConfig genesis.UnparsedConfig) error {
	var err error
	for i, s := range stakers {
		keyPath := fmt.Sprintf("%s/%s/staking/staker.key", BASE_DIR, s.NodeID)
		certPath := fmt.Sprintf("%s/%s/staking/staker.crt", BASE_DIR, s.NodeID)

		err = writeOutKeyAndCert(keyPath, s.KeyBytes, certPath, s.CertBytes)
		if err != nil {
			fmt.Printf("Write out staker.key/staker.cert failed on node %s: %s\n", s.NodeID, err)
		}

		ip := fmt.Sprintf("%s%d", NETWORK_ADDRESS, i+2)
		err = writeOutNodeConfig(s, uint64(i), BASE_DIR, ip, stakers[0].NodeID.String(), fmt.Sprintf("%s2", NETWORK_ADDRESS), genesisConfig)
		if err != nil {
			fmt.Printf("Write out node config failed on node %s: %s\n", s.NodeID, err)
		}
	}

	return nil
}

func writeOutKeyAndCert(keyPath string, keyBytes []byte, certPath string, certBytes []byte) error {
	// Ensure directory where key/cert will live exist
	if err := os.MkdirAll(filepath.Dir(certPath), perms.ReadWriteExecute); err != nil {
		return fmt.Errorf("couldn't create path for cert: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), perms.ReadWriteExecute); err != nil {
		return fmt.Errorf("couldn't create path for key: %w", err)
	}

	// Write cert to disk
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("couldn't create cert file: %w", err)
	}
	if _, err := certFile.Write(certBytes); err != nil {
		return fmt.Errorf("couldn't write cert file: %w", err)
	}
	if err := certFile.Close(); err != nil {
		return fmt.Errorf("couldn't close cert file: %w", err)
	}
	if err := os.Chmod(certPath, perms.ReadOnly); err != nil { // Make cert read-only
		return fmt.Errorf("couldn't change permissions on cert: %w", err)
	}

	// Write key to disk
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("couldn't create key file: %w", err)
	}
	if _, err := keyOut.Write(keyBytes); err != nil {
		return fmt.Errorf("couldn't write private key: %w", err)
	}
	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("couldn't close key file: %w", err)
	}
	if err := os.Chmod(keyPath, perms.ReadOnly); err != nil { // Make key read-only
		return fmt.Errorf("couldn't change permissions on key: %w", err)
	}

	return nil
}

func writeOutNodeConfig(s version1.Staker, index uint64, fileDir string, publicIp string, bootstrapNodeId string, bootstrapNodeIp string, genesisConfig genesis.UnparsedConfig) error {
	rootMntDir := "/mnt"

	var bootstrapIps string
	var bootstrapIds string
	if index > 0 {
		bootstrapIps = fmt.Sprintf("%s:9651", bootstrapNodeIp)
		bootstrapIds = bootstrapNodeId
	}
	config := &version1.NodeConfig{
		DataDir:         fmt.Sprintf("%s/node", rootMntDir),
		HttpPort:        9650,
		StakingPort:     9651,
		HttpHost:        "0.0.0.0",
		PublicIp:        publicIp,
		IndexEnabled:    true,
		ApiAdminEnabled: true,
		LogDisplayLevel: "INFO",
		LogLevel:        "DEBUG",
		NetworkID:       54321,
		BootstrapIPs:    bootstrapIps,
		BootstrapIDs:    bootstrapIds,
	}
	configJson, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}
	configPath := fmt.Sprintf("%s/%s/config.json", fileDir, s.NodeID)
	configFile, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer configFile.Close()
	_, err = configFile.Write(configJson)
	if err != nil {
		return err
	}

	genesisJson, err := json.MarshalIndent(genesisConfig, "", "\t")
	if err != nil {
		return err
	}
	genesisConfigPath := fmt.Sprintf("%s/%s/genesis.json", fileDir, s.NodeID)
	genesisFile, err := os.Create(genesisConfigPath)
	if err != nil {
		return err
	}
	defer genesisFile.Close()
	_, err = genesisFile.Write(genesisJson)
	if err != nil {
		return err
	}
	return nil
}

func CreateComposeFile(baseDir string, stakers []version1.Staker, image string) error {
	localNetworkConfig := [1]NetworkConfig{
		{
			Subnet:  fmt.Sprintf("%s0/24", NETWORK_ADDRESS),
			Gateway: fmt.Sprintf("%s1", NETWORK_ADDRESS),
		},
	}
	yml := &DockerComposeV3{
		Version:  "3",
		Services: make(map[string]Container),
		Networks: map[string]Network{
			"camino-local": {
				Driver: "bridge",
				Ipam:   struct{ Config []NetworkConfig }{Config: localNetworkConfig[:]},
			},
		},
	}

	for i, s := range stakers {
		volumes := [1]string{fmt.Sprintf("./%s:/mnt/node", s.NodeID)}
		ports := [1]string{fmt.Sprintf("%d:%d", 9650+i*2, 9650)}
		yml.Services[s.NodeID.String()] = Container{
			Image:      image,
			Entrypoint: "./camino-node --config-file /mnt/node/config.json --genesis /mnt/node/genesis.json",
			Volumes:    volumes[:],
			Ports:      ports[:],
			Networks: map[string]map[string]string{
				"camino-local": {
					"ipv4_address": fmt.Sprintf("%s%d", NETWORK_ADDRESS, i+2),
				},
			},
		}
	}

	content, err := yaml.Marshal(&yml)
	if err != nil {
		return err
	}
	f, err := os.Create(fmt.Sprintf("%s/docker-compose.yml", baseDir))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(content)
	if err != nil {
		return err
	}

	return nil
}
