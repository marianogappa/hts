package signaltranspiler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/marianogappa/signal-checker/common"
)

var instructions = []instruction{
	instrMarket{},
	instrEmpty{},
	instrEnterImmediately{},
	instrEnter{},
	instrTakeProfit{},
	instrStopLoss{},
	instrExchange{},
	instrInitialISO8601{},
	instrIsShort{},
	instrInvalidate{},
}

var (
	rxPair              = regexp.MustCompile(`^\s*(PAIR:?|SYMBOL:?|MARKET:?)?\s*([[:upper:]]{2,6})[/-]([[:upper:]]{2,6})\s*(//.*)?$`)
	rxEmpty             = regexp.MustCompile(`^\s*(//.*)?$`)
	rxEnterImmediately  = regexp.MustCompile(`^\s*(ENTER:?)\s*(NOW|IMMEDIATELY)\s*(//.*)?$`)
	rxEnter             = regexp.MustCompile(`^\s*(ENTER:?|ENTER AT:?|ENTER BETWEEN:?|ENTER RANGE:?)\s*(([\d.]+\s*(,|-|AND)?\s*)+?)\s*(//.*)?$`)
	rxTakeProfit        = regexp.MustCompile(`^\s*(TAKE PROFIT:?|TP:?)\s*(([\d.]+\s*[,-]?\s*)+?)\s*(//.*)?$`)
	rxStopLoss          = regexp.MustCompile(`^\s*(STOP LOSS:?|SL:?)?\s*([\d.]+)\s*(//.*)?$`)
	rxExchange          = regexp.MustCompile(`^\s*(EXCHANGE:?|PLATFORM:?)?\s*([[:upper:]]+)\s*(//.*)?$`)
	rxInitialISO8601    = regexp.MustCompile(`^\s*(START AT:?|INITIALISO8601:?|FROM:?|AT:?|START:?)?\s*(.+?)\s*(//.*)?$`)
	rxInvalidateISO8601 = regexp.MustCompile(`^\s*((TIMEOUT|INVALIDATE) (IN|AFTER|WITHIN):?)?\s*([\d]+?)\s+DAYS\s*(//.*)?$`)
	rxIsShort           = regexp.MustCompile(`^\s*(LONG|SHORT)\s*(//.*)?$`)

	exchangeList = []string{
		"BINANCE",
		"BINANCE FUTURES",
		"BINANCEUSDMFUTURES",
		"BINANCE USDM FUTURES",
		"HUOBI",
		"COINBASE",
		"KRAKEN",
		"KUCOIN",
		"BITHUMB",
		"BINANCE.US",
		"BITFINEX",
		"GATE.IO",
		"BITSTAMP",
		"COINONE",
		"BITFLYER",
		"GEMINI",
		"POLONIEX",
		"BITTREX",
		"OKEX",
		"LIQUID",
		"FTX",
		"COINCHECK",
		"KORBIT",
		"CRYPTO.COM",
		"UPBIT",
		"ASCENDEX",
		"BITMAX",
	}

	supportedExchangeList = []string{
		"BINANCE",
		"BINANCE FUTURES",
		"BINANCEUSDMFUTURES",
		"BINANCE USDM FUTURES",
		"COINBASE",
		"KRAKEN",
		"KUCOIN",
		"FTX",
	}
)

type instrMarket struct{}

func (si instrMarket) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxPair.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	if sto.SignalInput.BaseAsset != "" || sto.SignalInput.QuoteAsset != "" {
		return signalInstruction{
			err: fmt.Errorf("%w [%v]", errMarketAlreadySupplied, rawInput),
			tokenizedInput: []inputToken{
				{Input: rawInput, TokenType: TOKEN_ERROR},
			},
		}, true
	}
	sto.SignalInput.BaseAsset = result[2]
	sto.SignalInput.QuoteAsset = result[3]
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "MARKET", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: sto.SignalInput.BaseAsset, TokenType: TOKEN_EXPRESSION},
			{Input: "/", TokenType: TOKEN_PUNCTUATION},
			{Input: sto.SignalInput.QuoteAsset, TokenType: TOKEN_EXPRESSION},
		},
	}, true
}

type instrEmpty struct{}

func (si instrEmpty) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
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

type instrEnterImmediately struct{}

func (si instrEnterImmediately) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxEnterImmediately.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	sto.SignalInput.EnterRangeLow = -1
	sto.SignalInput.EnterRangeHigh = -1
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "ENTER", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: "IMMEDIATELY", TokenType: TOKEN_EXPRESSION},
		},
	}, true
}

type instrTakeProfit struct{}

func (si instrTakeProfit) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxTakeProfit.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	fls, err := extractFloatSequence(result[2])
	if err != nil {
		return signalInstruction{
			err: fmt.Errorf("%w with content %v", errMalformedFloat, result[2]),
			tokenizedInput: []inputToken{
				{Input: "TAKE PROFIT", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: result[2], TokenType: TOKEN_ERROR},
			},
		}, true
	}
	tokenizedInput := []inputToken{
		{Input: "TAKE PROFIT", TokenType: TOKEN_INSTRUCTION},
		{Input: ": ", TokenType: TOKEN_PUNCTUATION},
	}

	for _, fl := range fls {
		cfl := common.JsonFloat64(fl)
		cfls, _ := json.Marshal(cfl)
		tokenizedInput = append(tokenizedInput, inputToken{Input: string(cfls), TokenType: TOKEN_EXPRESSION})
		tokenizedInput = append(tokenizedInput, inputToken{Input: " ", TokenType: TOKEN_PUNCTUATION})
		sto.SignalInput.TakeProfits = append(sto.SignalInput.TakeProfits, cfl)
	}

	return signalInstruction{
		tokenizedInput: tokenizedInput,
	}, true
}

type instrEnter struct{}

func (si instrEnter) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxEnter.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	if sto.SignalInput.EnterRangeLow != common.JsonFloat64(0.0) || sto.SignalInput.EnterRangeHigh != common.JsonFloat64(0.0) {
		return signalInstruction{
			err: fmt.Errorf("%w [%v]", errEnterRangeAlreadySupplied, rawInput),
			tokenizedInput: []inputToken{
				{Input: rawInput, TokenType: TOKEN_ERROR},
			},
		}, true
	}
	fls, err := extractFloatSequence(result[2])
	if err != nil {
		return signalInstruction{
			err: fmt.Errorf("%w with content %v", errMalformedFloat, result[2]),
			tokenizedInput: []inputToken{
				{Input: "ENTER BETWEEN", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: result[2], TokenType: TOKEN_ERROR},
			},
		}, true
	}
	if len(fls) != 2 {
		return signalInstruction{
			err: fmt.Errorf("%w [%v], supply exactly two values e.g. ENTER BETWEEN: 0.1 - 0.5", errInvalidEnterAt, result[2]),
			tokenizedInput: []inputToken{
				{Input: "ENTER BETWEEN", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: result[2], TokenType: TOKEN_ERROR},
			},
		}, true
	}

	cfl1, cfl2 := common.JsonFloat64(fls[0]), common.JsonFloat64(fls[1])
	cfls1, _ := json.Marshal(cfl1)
	cfls2, _ := json.Marshal(cfl2)

	if fls[0] > fls[1] {
		return signalInstruction{
			err: fmt.Errorf("%w [%v], the second number in the range should be higher", errInvalidEnterRange, result[2]),
			tokenizedInput: []inputToken{
				{Input: "ENTER BETWEEN", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: string(cfls1), TokenType: TOKEN_EXPRESSION},
				{Input: " - ", TokenType: TOKEN_PUNCTUATION},
				{Input: string(cfls2), TokenType: TOKEN_ERROR},
			},
		}, true
	}

	sto.SignalInput.EnterRangeLow = cfl1
	sto.SignalInput.EnterRangeHigh = cfl2
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "ENTER BETWEEN", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: string(cfls1), TokenType: TOKEN_EXPRESSION},
			{Input: " - ", TokenType: TOKEN_PUNCTUATION},
			{Input: string(cfls2), TokenType: TOKEN_EXPRESSION},
		},
	}, true
}

type instrStopLoss struct{}

func (si instrStopLoss) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxStopLoss.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	if sto.SignalInput.StopLoss != common.JsonFloat64(0.0) {
		return signalInstruction{
			err: fmt.Errorf("%w [%v]", errStopLossAlreadySupplied, rawInput),
			tokenizedInput: []inputToken{
				{Input: rawInput, TokenType: TOKEN_ERROR},
			},
		}, true
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
	sto.SignalInput.StopLoss = common.JsonFloat64(fl)
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "STOP LOSS", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: maybeFloat, TokenType: TOKEN_EXPRESSION},
		},
	}, true
}

type instrExchange struct{}

func (si instrExchange) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxExchange.FindStringSubmatch(upRawInput)
	if len(result) == 0 {
		return signalInstruction{}, false
	}
	if result[1] == "" && !isSInSS(result[2], exchangeList) {
		return signalInstruction{}, false
	}
	if result[1] != "" && !isSInSS(result[2], supportedExchangeList) {
		return signalInstruction{
			err: fmt.Errorf("%w for exchange [%v]", errUnsupportedExchange, result[2]),
			tokenizedInput: []inputToken{
				{Input: "EXCHANGE", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: result[2], TokenType: TOKEN_ERROR},
			},
		}, true
	}
	if sto.SignalInput.Exchange != "" {
		return signalInstruction{
			err: fmt.Errorf("%w [%v]", errExchangeAlreadySupplied, rawInput),
			tokenizedInput: []inputToken{
				{Input: rawInput, TokenType: TOKEN_ERROR},
			},
		}, true
	}
	sto.SignalInput.Exchange = strings.ToLower(result[2])
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "EXCHANGE", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: result[2], TokenType: TOKEN_EXPRESSION},
		},
	}, true
}

type instrInitialISO8601 struct{}

func (si instrInitialISO8601) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxInitialISO8601.FindStringSubmatch(upRawInput)

	// TODO support other formats
	iso8601, err := tryParseDate(result[2])
	if result[1] == "" && err != nil {
		return signalInstruction{}, false
	}
	if sto.SignalInput.InitialISO8601 != "" {
		return signalInstruction{
			err: fmt.Errorf("%w [%v]", errInitialISO8601AlreadySupplied, rawInput),
			tokenizedInput: []inputToken{
				{Input: rawInput, TokenType: TOKEN_ERROR},
			},
		}, true
	}
	if result[1] != "" && err != nil {
		return signalInstruction{
			err: fmt.Errorf("%w for datetime [%v]", errUnsupportedDateTimeFormat, result[2]),
			tokenizedInput: []inputToken{
				{Input: "START AT", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: result[2], TokenType: TOKEN_ERROR},
			},
		}, true
	}
	sto.SignalInput.InitialISO8601 = common.ISO8601(iso8601)
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "START AT", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: result[2], TokenType: TOKEN_EXPRESSION},
		},
	}, true
}

type instrIsShort struct{}

func (si instrIsShort) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxIsShort.FindStringSubmatch(upRawInput)

	if len(result) == 0 {
		return signalInstruction{}, false
	}
	if sto.isShortSet {
		return signalInstruction{
			err: fmt.Errorf("%w [%v]", errIsShortAlreadySupplied, rawInput),
			tokenizedInput: []inputToken{
				{Input: rawInput, TokenType: TOKEN_ERROR},
			},
		}, true
	}
	sto.isShortSet = true

	if result[1] == "SHORT" {
		sto.SignalInput.IsShort = true
		return signalInstruction{
			tokenizedInput: []inputToken{
				{Input: "SHORT", TokenType: TOKEN_INSTRUCTION},
			},
		}, true
	}
	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "LONG", TokenType: TOKEN_INSTRUCTION},
		},
	}, true
}

type instrInvalidate struct{}

func (si instrInvalidate) apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool) {
	upRawInput := strings.ToUpper(rawInput)
	result := rxInvalidateISO8601.FindStringSubmatch(upRawInput)

	if len(result) == 0 {
		return signalInstruction{}, false
	}
	if sto.SignalInput.InvalidateAfterSeconds > 0 {
		return signalInstruction{
			err: fmt.Errorf("%w [%v]", errInvalidateAfterDaysAlreadySupplied, rawInput),
			tokenizedInput: []inputToken{
				{Input: rawInput, TokenType: TOKEN_ERROR},
			},
		}, true
	}

	days, err := strconv.Atoi(result[4])
	if err != nil {
		return signalInstruction{
			err: fmt.Errorf("%w [%v]", errMalformedInteger, result[4]),
			tokenizedInput: []inputToken{
				{Input: "TIMEOUT AFTER", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: result[4], TokenType: TOKEN_ERROR},
				{Input: " DAYS", TokenType: TOKEN_EXPRESSION},
			},
		}, true
	}
	if days > 7 {
		return signalInstruction{
			err: fmt.Errorf("%w [%v]", errMaximumInvalidation7Days, result[4]),
			tokenizedInput: []inputToken{
				{Input: "TIMEOUT AFTER", TokenType: TOKEN_INSTRUCTION},
				{Input: ": ", TokenType: TOKEN_PUNCTUATION},
				{Input: fmt.Sprintf("%v", days), TokenType: TOKEN_ERROR},
				{Input: " DAYS", TokenType: TOKEN_EXPRESSION},
			},
		}, true
	}

	sto.SignalInput.InvalidateAfterSeconds = days * 86400

	return signalInstruction{
		tokenizedInput: []inputToken{
			{Input: "TIMEOUT AFTER", TokenType: TOKEN_INSTRUCTION},
			{Input: ": ", TokenType: TOKEN_PUNCTUATION},
			{Input: fmt.Sprintf("%v", days), TokenType: TOKEN_EXPRESSION},
			{Input: " DAYS", TokenType: TOKEN_EXPRESSION},
		},
	}, true
}

func isSInSS(s string, ss []string) bool {
	for _, si := range ss {
		if s == si {
			return true
		}
	}
	return false
}

func extractFloatSequence(fls string) ([]float64, error) {
	result := []float64{}

	containsComma := strings.Contains(fls, ",")
	containsDash := strings.Contains(fls, "-")
	containsAnd := strings.Contains(fls, "AND")
	if (containsComma && containsDash) || (containsComma && containsAnd) || (containsDash && containsAnd) {
		return result, errMixesSeparators
	}
	flss := []string{fls}
	if containsComma {
		flss = strings.Split(fls, ",")
	}
	if containsDash {
		flss = strings.Split(fls, "-")
	}
	if containsAnd {
		flss = strings.Split(fls, "AND")
	}
	for _, fls := range flss {
		for _, fl := range strings.Fields(fls) {
			f, err := strconv.ParseFloat(fl, 64)
			if err != nil {
				return result, errMalformedFloat
			}
			result = append(result, f)
		}
	}
	return result, nil
}

func tryParseDate(s string) (common.ISO8601, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02",
	}
	var (
		t   time.Time
		err error
	)
	for _, format := range formats {
		t, err = time.Parse(format, s)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", err
	}
	return common.ISO8601(t.Format(time.RFC3339)), nil
}
