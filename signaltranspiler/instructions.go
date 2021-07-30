package signaltranspiler

import (
	"regexp"
	"strings"

	"github.com/marianogappa/signal-checker/common"
)

var instructions = []instruction{
	instrMarket{},
}

type instrMarket struct{}

var (
	rxPair = regexp.MustCompile(`^\s*(PAIR:?|SYMBOL:?|MARKET:?)?\s*([[:upper:]]{2,6})[/-]([[:upper:]]{2,6})\s*$`)
)

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
