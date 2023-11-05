package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
	subsciptiontypes "github.com/lavanet/lava/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) MonthlyPayout(goCtx context.Context, req *types.QueryMonthlyPayoutRequest) (*types.QueryMonthlyPayoutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var amount uint64

	// get all tracked CU entries
	trackedCuInds := k.subscriptionKeeper.GetAllSubTrackedCuIndices(ctx, "")

	type totalUsedCuInfo struct {
		totalUsedCu    uint64
		relevant       bool // sub is relevant if the provider provided service for it
		block          uint64
		providerCuInfo map[string]uint64 // map[chainID]providerUsedCu
	}

	// get a map of sub+chainID to properties for reward calculation
	totalUsedCuMap := map[string]totalUsedCuInfo{}
	for _, ind := range trackedCuInds {
		sub, provider, chainID := subsciptiontypes.DecodeCuTrackerKey(ind)
		subObj, found := k.subscriptionKeeper.GetSubscription(ctx, sub)
		if !found {
			return nil, utils.LavaFormatError("cannot get tracked CU", fmt.Errorf("subscription not found"),
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chain_id", Value: chainID},
			)
		}

		subBlock := subObj.Block
		cu, found, _ := k.subscriptionKeeper.GetTrackedCu(ctx, sub, provider, chainID, subBlock)
		if found {
			// check if sub got service from provider (mark relevant and keep the provider's CU)
			relevant := false
			var providerCu uint64
			if provider == req.Provider {
				relevant = true
				providerCu = cu
			}

			// count CU (we also count CU of sub that is not relevant because it might got service from many
			// providers, and one of them might be the requested provider)
			key := sub
			if usedCuInfo, ok := totalUsedCuMap[key]; ok {
				usedCuInfo.totalUsedCu += cu
				usedCuInfo.relevant = relevant
				usedCuInfo.block = subBlock
				if providerCu != 0 {
					providerCuInfoMap := usedCuInfo.providerCuInfo
					providerCuInfoMap[chainID] = providerCu
					usedCuInfo.providerCuInfo = providerCuInfoMap
				}

				totalUsedCuMap[key] = usedCuInfo
			} else {
				totalUsedCuMap[key] = totalUsedCuInfo{
					totalUsedCu:    cu,
					relevant:       relevant,
					block:          subBlock,
					providerCuInfo: map[string]uint64{},
				}
				if providerCu != 0 {
					providerCuMap := map[string]uint64{chainID: providerCu}
					info := totalUsedCuMap[key]
					info.providerCuInfo = providerCuMap
					totalUsedCuMap[key] = info
				}
			}
		}
	}

	// calculate the provider's reward
	for sub, usedCuInfo := range totalUsedCuMap {
		if usedCuInfo.relevant {
			plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, sub, usedCuInfo.block)
			if err != nil {
				return nil, err
			}

			for chainID, providerCu := range usedCuInfo.providerCuInfo {
				// totalMonthlyReward = providerReward + totalDelegatorsReward
				totalMonthlyReward := k.subscriptionKeeper.CalcTotalMonthlyReward(ctx, plan, providerCu, usedCuInfo.totalUsedCu)

				providerAddr, err := sdk.AccAddressFromBech32(req.Provider)
				if err != nil {
					return nil, utils.LavaFormatError("invalid provider address", err,
						utils.Attribute{Key: "provider", Value: req.Provider},
					)
				}

				// calculate only the provider reward
				providerReward, err := k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, chainID, totalMonthlyReward, subsciptiontypes.ModuleName, true)
				if err != nil {
					return nil, err
				}

				amount += providerReward.Uint64()
			}
		}
	}

	return &types.QueryMonthlyPayoutResponse{Amount: amount}, nil
}
