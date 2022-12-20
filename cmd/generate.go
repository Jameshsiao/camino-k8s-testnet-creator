/*
 * generate.go
 * Copyright (C) 2022, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"chain4travel.com/camktncr/pkg/version1"
	"chain4travel.com/camktncr/pkg/version1/dockercompose"
	"github.com/spf13/cobra"
)

func init() {

	generateCmd.Flags().Uint64("num-stakers", 20, "number of stakers total")
	generateCmd.Flags().Uint64("num-initial-stakers", 5, "number of initial stakers")
	generateCmd.Flags().Uint64("default-stake", 2e5, "initial stake for each validator")
	generateCmd.Flags().Bool("override", false, "overwrite and delete existing data")

	// docker-compose custom local
	generateCmd.Flags().Bool("docker-compose", false, "generate docker-compose instead of k8s")
	generateCmd.Flags().String("image", "c4tplatform/camino-node:chain4travel", "docker image for node container")
	generateCmd.Flags().Uint64("num-archive-nodes", 0, "number of archive nodes")
}

const DENOMINATION = 1e9

var generateCmd = &cobra.Command{
	Use:   "generate <network-name>",
	Short: "generates a network with the specified config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		networkName := args[0]

		override, err := cmd.Flags().GetBool("override")
		if err != nil {
			return err
		}
		isDockerCompose, err := cmd.Flags().GetBool("docker-compose")
		if err != nil {
			return err
		}

		networkPath := fmt.Sprintf("%s.json", networkName)
		_, err = os.Stat(networkPath)
		if err == nil && !override {
			return fmt.Errorf("will not override existing data without --overide flag")
		}

		defaultStake, err := cmd.Flags().GetUint64("default-stake")
		if err != nil {
			return err
		}
		numStakers, err := cmd.Flags().GetUint64("num-stakers")
		if err != nil {
			return err
		}
		numInitialStakers, err := cmd.Flags().GetUint64("num-initial-stakers")
		if err != nil {
			return err
		}
		numArchiveNodes, err := cmd.Flags().GetUint64("num-archive-nodes")
		if err != nil {
			return err
		}

		networkId := 1002
		if isDockerCompose {
			networkId = version1.DOCKER_COMPOSE_LOCAL_NETWORK_ID
		}
		networkConfig := version1.NetworkConfig{
			NumStakers:        numStakers,
			NetworkID:         uint64(networkId),
			NetworkName:       networkName,
			DefaultStake:      defaultStake * DENOMINATION,
			NumInitialStakers: numInitialStakers,
		}

		now := uint64(time.Now().Unix())
		network, err := version1.BuildNetwork(networkConfig, now)
		if err != nil {
			return err
		}

		networkJson, err := json.MarshalIndent(network, "", "\t")
		if err != nil {
			return err
		}

		err = os.WriteFile(networkPath, networkJson, 0700)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}

		// Docker-compose custom local
		if isDockerCompose {
			image, err := cmd.Flags().GetString("image")
			if err != nil {
				return err
			}
			err = dockercompose.CreateComposeFiles(network.Stakers, network.GenesisConfig, image, numArchiveNodes)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
