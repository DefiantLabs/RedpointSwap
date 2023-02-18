package endpoints

import (
	"net/http"

	"github.com/DefiantLabs/RedpointSwap/api"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gin-gonic/gin"
)

type StatusRequest struct {
	Id string `json:"id"`
}

type TradeStatus struct {
	Error               string //if there is some error getting status for the ID
	UserTxStatus        string
	UserArbitrageStatus string
	UserSwaps           []api.Swap //The user's swaps (for their 'normal' trade)
	UserArb             []sdk.Coin //Arb we sent to the user ('ArbitrageStatus' will indicate received, pending, or failed)
}

func GetTradeStatus(context *gin.Context) {
	var request StatusRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}
	ats, err := api.GetStatusForSubmittedTxs(request.Id)
	if err != nil {
		context.JSON(http.StatusOK, &TradeStatus{Error: err.Error()})
		return
	}

	tradeStatus := convertToStatus(ats)
	context.JSON(http.StatusOK, tradeStatus)
}

func convertToStatus(userTrade *api.ArbitrageTxSet) TradeStatus {
	ts := TradeStatus{}
	ts.UserSwaps = getUserSwaps(userTrade)

	if userTrade.Processed {
		ts.UserTxStatus = "Trade finished"
	} else {
		ts.UserTxStatus = "Trade submitted, waiting for chain"
	}

	if userTrade.UserProfitShareTx.Initiated && !userTrade.UserProfitShareTx.Committed {
		if userTrade.UserProfitShareTx.ArbitrageProfitsPending.IsZero() {
			ts.UserArbitrageStatus = "No arbitrage"
		} else {
			ts.UserArbitrageStatus = "Sent user arbitrage, waiting for chain"
			ts.UserArb = userTrade.UserProfitShareTx.ArbitrageProfitsPending
		}
	}

	if userTrade.UserProfitShareTx.Initiated && userTrade.UserProfitShareTx.Committed {
		ts.UserArbitrageStatus = "User received arbitrage"
		ts.UserArb = userTrade.UserProfitShareTx.ArbitrageProfitsReceived

		if !userTrade.UserProfitShareTx.Succeeded {
			ts.UserArbitrageStatus = "Problem sending user arbitrage (will not reattempt, please report address and time of trade)"
		}
	}

	return ts
}

func getUserSwaps(userTrade *api.ArbitrageTxSet) []api.Swap {
	swaps := []api.Swap{}
	for _, t := range userTrade.TradeTxs {
		for _, swap := range t.Swaps {
			if swap.IsUserSwap {
				swaps = append(swaps, swap)
			}
		}
	}

	return swaps
}