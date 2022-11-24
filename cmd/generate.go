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

	// docker-compose local
	generateCmd.Flags().String("image", "c4tplatform/camino-node:chain4travel", "docker image for node container")
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

		networkId := 1002
		if networkName == "local" {
			networkId = 54321
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

		// Docker-compose local
		if networkName == "local" {
			image, err := cmd.Flags().GetString("image")
			if err != nil {
				return err
			}
			err = dockercompose.CreateComposeFile(network.Stakers, network.GenesisConfig, image)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
