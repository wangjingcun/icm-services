// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validators

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/validators"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	avalancheWarp "github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/awm-relayer/config"
	"github.com/ava-labs/awm-relayer/peers/utils"
	"go.uber.org/zap"

	pchainapi "github.com/ava-labs/avalanchego/vms/platformvm/api"
)

var _ validators.State = &CanonicalValidatorClient{}

// CanonicalValidatorClient wraps platformvm.Client and implements validators.State
type CanonicalValidatorClient struct {
	logger  logging.Logger
	client  platformvm.Client
	options []rpc.Option
}

func NewCanonicalValidatorClient(logger logging.Logger, apiConfig *config.APIConfig) *CanonicalValidatorClient {
	client := platformvm.NewClient(apiConfig.BaseURL)
	options := utils.InitializeOptions(apiConfig)
	return &CanonicalValidatorClient{
		logger:  logger,
		client:  client,
		options: options,
	}
}

func (v *CanonicalValidatorClient) GetCurrentCanonicalValidatorSet(
	subnetID ids.ID,
) ([]*avalancheWarp.Validator, uint64, error) {
	// Get the current canonical validator set of the source subnet.
	canonicalSubnetValidators, totalValidatorWeight, err := avalancheWarp.GetCanonicalValidatorSet(
		context.Background(),
		v,
		pchainapi.ProposedHeight,
		subnetID,
	)
	if err != nil {
		v.logger.Error(
			"Failed to get the canonical subnet validator set",
			zap.String("subnetID", subnetID.String()),
			zap.Error(err),
		)
		return nil, 0, err
	}

	return canonicalSubnetValidators, totalValidatorWeight, nil
}

func (v *CanonicalValidatorClient) GetMinimumHeight(ctx context.Context) (uint64, error) {
	return v.client.GetHeight(ctx, v.options...)
}

func (v *CanonicalValidatorClient) GetCurrentHeight(ctx context.Context) (uint64, error) {
	return v.client.GetHeight(ctx, v.options...)
}

func (v *CanonicalValidatorClient) GetSubnetID(ctx context.Context, blockchainID ids.ID) (ids.ID, error) {
	return v.client.ValidatedBy(ctx, blockchainID, v.options...)
}

// Not called directly just defined for interface implementation. see private method getCurrentValidatorSet
// for actual helper in case [platform.GetValidatorsAt] call fails
func (v *CanonicalValidatorClient) GetCurrentValidatorSet(
	_ context.Context,
	_ ids.ID,
) (map[ids.ID]*validators.GetCurrentValidatorOutput, uint64, error) {
	return nil, 0, nil
}

// Gets the validator set of the given subnet at the given P-chain block height.
// Attempts to use the "getValidatorsAt" API first. If not available, falls back
// to use "getCurrentValidators", ignoring the specified P-chain block height.
func (v *CanonicalValidatorClient) GetValidatorSet(
	ctx context.Context,
	height uint64,
	subnetID ids.ID,
) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	// First, attempt to use the "getValidatorsAt" RPC method. This method may not be available on
	// all API nodes, in which case we can fall back to using "getCurrentValidators" if needed.
	res, err := v.client.GetValidatorsAt(ctx, subnetID, pchainapi.Height(height), v.options...)
	if err != nil {
		v.logger.Debug(
			"P-chain RPC to getValidatorAt returned error. Falling back to getCurrentValidators",
			zap.String("subnetID", subnetID.String()),
			zap.Uint64("pChainHeight", height),
			zap.Error(err))
		return nil, err
	}
	return res, nil
}
