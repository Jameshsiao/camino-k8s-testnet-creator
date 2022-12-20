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
	COMPOSE_DIR     = "local/docker-compose"
	NETWORK_ADDRESS = "10.0.7."
	ROOT_MNT_DIR    = "/mnt"
)

func createCertFileAndNodeConfig(stakers []version1.Staker, genesisConfig genesis.UnparsedConfig) error {
	var err error
	for i, s := range stakers {
		keyPath := fmt.Sprintf("%s/%s/staking/staker.key", COMPOSE_DIR, s.NodeID)
		certPath := fmt.Sprintf("%s/%s/staking/staker.crt", COMPOSE_DIR, s.NodeID)

		err = writeOutKeyAndCert(keyPath, s.KeyBytes, certPath, s.CertBytes)
		if err != nil {
			return fmt.Errorf("write out staker.key/staker.cert failed on node %s: %w", s.NodeID, err)
		}

		err = writeOutNodeConfig(s.NodeID.String(), uint64(i), stakers[0].NodeID.String(), fmt.Sprintf("%s2", NETWORK_ADDRESS), genesisConfig)
		if err != nil {
			return fmt.Errorf("write out node config failed on node %s: %w", s.NodeID, err)
		}

		cChainConfig := version1.CChainConfig{
			PruningEnabled:              true,
			AllowMissingTries:           true,
			OfflinePruningEnabled:       false,
			OfflinePruningDataDirectory: fmt.Sprintf("%s/node/offline-pruning", ROOT_MNT_DIR),
		}
		err = writeOutCChainConfig(s.NodeID.String(), cChainConfig)
		if err != nil {
			return fmt.Errorf("write out C-Chain config failed on node %s: %w", s.NodeID, err)
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

func writeOutNodeConfig(nodeID string, index uint64, bootstrapNodeId string, bootstrapNodeIp string, genesisConfig genesis.UnparsedConfig) error {
	var bootstrapIps string
	var bootstrapIds string
	if index > 0 {
		bootstrapIps = fmt.Sprintf("%s:9651", bootstrapNodeIp)
		bootstrapIds = bootstrapNodeId
	}
	publicIp := fmt.Sprintf("%s%d", NETWORK_ADDRESS, index+2)
	config := &version1.NodeConfig{
		DataDir:         fmt.Sprintf("%s/node", ROOT_MNT_DIR),
		HttpPort:        9650,
		StakingPort:     9651,
		HttpHost:        "0.0.0.0",
		PublicIp:        publicIp,
		IndexEnabled:    true,
		ApiAdminEnabled: true,
		LogDisplayLevel: "TRACE",
		LogLevel:        "DEBUG",
		NetworkID:       54321,
		BootstrapIPs:    bootstrapIps,
		BootstrapIDs:    bootstrapIds,
	}
	configJson, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}
	configPath := fmt.Sprintf("%s/%s/config.json", COMPOSE_DIR, nodeID)
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
	genesisConfigPath := fmt.Sprintf("%s/%s/genesis.json", COMPOSE_DIR, nodeID)
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

func writeOutCChainConfig(nodeID string, cChainConfig version1.CChainConfig) error {
	cChainConfigPath := fmt.Sprintf("%s/%s/chains/C/config.json", COMPOSE_DIR, nodeID)
	// Ensure directory where C-chain config file will live exist
	if err := os.MkdirAll(filepath.Dir(cChainConfigPath), perms.ReadWriteExecute); err != nil {
		return fmt.Errorf("couldn't create path for C-chain config: %w", err)
	}
	// Write C-chain config to disk
	cChainConfigFile, err := os.Create(cChainConfigPath)
	if err != nil {
		return fmt.Errorf("couldn't create C-chain config file: %w", err)
	}
	defer cChainConfigFile.Close()
	configJson, err := json.MarshalIndent(cChainConfig, "", "\t")
	if err != nil {
		return err
	}
	_, err = cChainConfigFile.Write(configJson)
	if err != nil {
		return err
	}

	return nil
}

func createArchiveNodeConfig(numArchiveNodes uint64, stakers []version1.Staker, genesisConfig genesis.UnparsedConfig) error {
	for i := 0; i < int(numArchiveNodes); i++ {
		archiveNodeId := fmt.Sprintf("Archive-Node-%d", i)
		cChainConfig := version1.CChainConfig{
			PruningEnabled: false,
		}
		err := writeOutCChainConfig(archiveNodeId, cChainConfig)
		if err != nil {
			return fmt.Errorf("couldn't create C-Chain config of archive node %s: %w", archiveNodeId, err)
		}

		// Write node config to disk
		err = writeOutNodeConfig(archiveNodeId, uint64(i+len(stakers)), stakers[0].NodeID.String(), fmt.Sprintf("%s2", NETWORK_ADDRESS), genesisConfig)
		if err != nil {
			return fmt.Errorf("couldn't write out node config on archive node %s: %w", archiveNodeId, err)
		}
	}

	return nil
}

func CreateComposeFiles(stakers []version1.Staker, genesisConfig genesis.UnparsedConfig, image string, numArchiveNodes uint64) error {
	if err := os.MkdirAll(filepath.Dir(COMPOSE_DIR), perms.ReadWriteExecute); err != nil {
		return fmt.Errorf("couldn't create compose dir %s: %w", COMPOSE_DIR, err)
	}

	err := createCertFileAndNodeConfig(stakers, genesisConfig)
	if err != nil {
		return fmt.Errorf("couldn't create cert file and node config: %w", err)
	}

	err = createArchiveNodeConfig(numArchiveNodes, stakers, genesisConfig)
	if err != nil {
		return fmt.Errorf("couldn't create C-chain config files for archive nodes: %w", err)
	}

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

	// Validators
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

	// Archive nodes
	for i := 0; i < int(numArchiveNodes); i++ {
		archiveNodeId := fmt.Sprintf("Archive-Node-%d", i)
		volumes := [1]string{fmt.Sprintf("./%s:/mnt/node", archiveNodeId)}
		ports := [1]string{fmt.Sprintf("%d:%d", 9650+(i+len(stakers))*2, 9650)}
		yml.Services[archiveNodeId] = Container{
			Image:      image,
			Entrypoint: "./camino-node --config-file /mnt/node/config.json --genesis /mnt/node/genesis.json --chain-config-dir /mnt/node/chains",
			Volumes:    volumes[:],
			Ports:      ports[:],
			Networks: map[string]map[string]string{
				"camino-local": {
					"ipv4_address": fmt.Sprintf("%s%d", NETWORK_ADDRESS, i+len(stakers)+2),
				},
			},
		}
	}

	content, err := yaml.Marshal(&yml)
	if err != nil {
		return err
	}
	f, err := os.Create(fmt.Sprintf("%s/docker-compose.yml", COMPOSE_DIR))
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
