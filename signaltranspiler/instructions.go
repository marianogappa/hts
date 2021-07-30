package signaltranspiler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/marianogappa/signal-checker/common"
)

var instructions = []instruction{
	instrMarket{},
	instrEmpty{},
	instrTakeProfit{},
	instrStopLoss{},
}

var (
	rxPair       = regexp.MustCompile(`^\s*(PAIR:?|SYMBOL:?|MARKET:?)?\s*([[:upper:]]{2,6})[/-]([[:upper:]]{2,6})\s*$`)
	rxEmpty      = regexp.MustCompile(`^\s*$`)
	rxTakeProfit = regexp.MustCompile(`^\s*(TAKE PROFIT:?|TP:?)?\s*([\d.,]+)\s*$`)
	rxStopLoss   = regexp.MustCompile(`^\s*(STOP LOSS:?|SL:?)?\s*([\d.,]+)\s*$`)
)

type instrMarket struct{}

func (si instrMarket) apply(rawInput string, signalInput *common.SignalCheckInput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxPair.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	signalInput.BaseAsset = result[2]
	signalInput.QuoteAsset = result[3]
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "MARKET", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: signalInput.BaseAsset, TokenType: TOKEN_EXPRESSION},
			{Input: "/", TokenType: TOKEN_PUNCTUATION},
			{Input: signalInput.QuoteAsset, TokenType: TOKEN_EXPRESSION},
		},
	}, true
}

type instrEmpty struct{}

func (si instrEmpty) apply(rawInput string, signalInput *common.SignalCheckInput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxEmpty.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: " ", TokenType: TOKEN_PUNCTUATION},
		},
	}, true
}

type instrTakeProfit struct{}

func (si instrTakeProfit) apply(rawInput string, signalInput *common.SignalCheckInput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxTakeProfit.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	maybeFloat := strings.ReplaceAll(result[2], ",", "")
	fl, err := strconv.ParseFloat(maybeFloat, 64)
	if err != nil {
		return signalInstruction{
			err: fmt.Errorf("%w with content %v", errMalformedFloat, result[2]),
			tokenizedInput: []inputToken{
				{Input: "TAKE PROFIT", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: maybeFloat, TokenType: TOKEN_ERROR},
			},
		}, true
	}
	signalInput.TakeProfits = []common.JsonFloat64{common.JsonFloat64(fl)}
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "TAKE PROFIT", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: maybeFloat, TokenType: TOKEN_EXPRESSION},
		},
	}, true
}

type instrStopLoss struct{}

func (si instrStopLoss) apply(rawInput string, signalInput *common.SignalCheckInput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxStopLoss.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	maybeFloat := strings.ReplaceAll(result[2], ",", "")
	fl, err := strconv.ParseFloat(maybeFloat, 64)
	if err != nil {
		return signalInstruction{
			err: fmt.Errorf("%w with content %v", errMalformedFloat, result[2]),
			tokenizedInput: []inputToken{
				{Input: "STOP LOSS", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: maybeFloat, TokenType: TOKEN_ERROR},
			},
		}, true
	}
	signalInput.StopLoss = common.JsonFloat64(fl)
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "STOP LOSS", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: maybeFloat, TokenType: TOKEN_EXPRESSION},
		},
	}, true
}
