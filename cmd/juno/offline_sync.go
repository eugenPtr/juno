package main

import (
	"fmt"
	"time"

	"github.com/NethermindEth/juno/blockchain"
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/juno/db/pebble"
	"github.com/NethermindEth/juno/utils"
	"github.com/spf13/cobra"
)

func OfflineSync() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "offline-sync",
		Short: "Offline sync.",
		RunE:  offlineSync,
	}

	cmd.Flags().String(dbPathF, "", "Path to the node database")
	cmd.Flags().String("sync-from", "", "Path to the feeder database")
	cmd.Flags().Uint64("block-num", 0, "Block number to sync to")
	return cmd
}

func offlineSync(cmd *cobra.Command, _ []string) error {
	// get the db path from the flags
	feederDBPath, err := cmd.Flags().GetString("sync-from")
	if err != nil {
		return err
	}

	feederDB, err := pebble.NewWithOptions(feederDBPath, 5000, 10000000, false)
	if err != nil {
		return err
	}

	feederBc := blockchain.New(feederDB, &utils.Sepolia)

	nodeDBPath, err := cmd.Flags().GetString(dbPathF)
	if err != nil {
		return err
	}

	nodeDB, err := pebble.NewWithOptions(nodeDBPath, 5000, 10000000, false)
	if err != nil {
		return err
	}

	target, err := cmd.Flags().GetUint64("block-num")
	if err != nil {
		return err
	}

	nodeBc := blockchain.New(nodeDB, &utils.Sepolia)
	startTime := time.Now()
	for i := uint64(0); i < target+1; i++ {
		lastTime := time.Now()
		block, err := feederBc.BlockByNumber(i)
		if err != nil {
			return err
		}

		stateUpdate, err := feederBc.StateUpdateByNumber(i)
		if err != nil {
			return err
		}

		reader, _, err := feederBc.StateAtBlockNumber(i)
		if err != nil {
			return err
		}

		newClasses, err := fetchUnknownClasses(reader, stateUpdate)
		if err != nil {
			return err
		}

		commitments, err := nodeBc.SanityCheckNewHeight(block, stateUpdate, newClasses)
		if err != nil {
			return err
		}

		err = nodeBc.Store(block, commitments, stateUpdate, newClasses)
		if err != nil {
			return err
		}

		fmt.Printf("Synced block %d, time taken: %s\n", i, time.Since(lastTime))
	}

	totalTime := time.Since(startTime)
	fmt.Printf("Total time taken: %s\n", totalTime)

	return nil
}

func fetchUnknownClasses(reader core.StateReader, stateUpdate *core.StateUpdate) (map[felt.Felt]core.Class, error) {
	newClasses := make(map[felt.Felt]core.Class)
	for _, classHash := range stateUpdate.StateDiff.DeployedContracts {
		class, err := reader.Class(classHash)
		if err != nil {
			return nil, err
		}

		newClasses[*classHash] = class.Class
	}

	for _, classHash := range stateUpdate.StateDiff.DeclaredV0Classes {
		if _, ok := newClasses[*classHash]; !ok {
			class, err := reader.Class(classHash)
			if err != nil {
				return nil, err
			}

			newClasses[*classHash] = class.Class
		}
	}

	for classHash := range stateUpdate.StateDiff.DeclaredV1Classes {
		if _, ok := newClasses[classHash]; !ok {
			class, err := reader.Class(&classHash)
			if err != nil {
				return nil, err
			}

			newClasses[classHash] = class.Class
		}
	}

	return newClasses, nil
}
