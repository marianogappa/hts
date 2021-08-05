package signaltranspiler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marianogappa/signal-checker/common"
)

type SignalTranspiler struct {
}

func NewSignalTranspiler() *SignalTranspiler {
	return &SignalTranspiler{}
}

type SignalTranspilerOutput struct {
	Errors         []string                 `json:"errors"`
	Warnings       []string                 `json:"warnings"`
	TokenizedInput [][]inputToken           `json:"tokenizedInput"`
	SignalInput    common.SignalCheckInput  `json:"signalInput"`
	SignalOutput   common.SignalCheckOutput `json:"signalOutput"`
	isShortSet     bool
}

func (o SignalTranspilerOutput) error() error {
	if len(o.Errors) == 0 {
		return nil
	}
	// TODO
	return errors.New("there were errors transpiling")
}

func (t SignalTranspiler) Transpile(input string) (SignalTranspilerOutput, error) {
	signalInstructions := []*signalInstruction{}
	for i, line := range strings.Split(input, "\n") {
		signalInstructions = append(signalInstructions, newSignalInstruction(line, i, false))
	}
	output := SignalTranspilerOutput{
		SignalInput: common.SignalCheckInput{ReturnCandlesticks: true},
		Errors:      []string{},
		Warnings:    []string{},
	}
	// 1. Instructions may be deferred, so do passes until the number of transpiled instructions is 0
	// 2. Do an inference pass (i.e. apply defaults)
	// 3. Do passes again until the number of transpiled instructions is 0
	// 4. Do a validation pass: if required params are missing, fail. If there are still untranspiled instructions, fail.
	for _, signalInstruction := range signalInstructions {
		var err error
		output, err = signalInstruction.apply(output)
		if err != nil {
			output.Errors = append(output.Errors, err.Error())
			continue
		}
	}

	signalInstructions = append(signalInstructions, t.calculateInferredInstructions(output)...)

	for _, signalInstruction := range signalInstructions {
		var err error
		output, err = signalInstruction.apply(output)
		if err != nil {
			output.Errors = append(output.Errors, err.Error())
			continue
		}
	}

	t.calculateErrorsAndWarnings(&output)

	for _, signalInstruction := range signalInstructions {
		if len(signalInstruction.tokenizedInput) == 0 {
			output.TokenizedInput = append(output.TokenizedInput, []inputToken{{Input: signalInstruction.rawInput, TokenType: TOKEN_ERROR}})
			continue
		}
		output.TokenizedInput = append(output.TokenizedInput, signalInstruction.tokenizedInput)
	}

	return output, output.error()
}

func (t SignalTranspiler) calculateInferredInstructions(sto SignalTranspilerOutput) []*signalInstruction {
	inferredInstructions := []*signalInstruction{}
	if sto.SignalInput.Exchange == "" {
		inferredInstructions = append(inferredInstructions, newSignalInstruction("EXCHANGE: binance", 0, true))
	}
	if !sto.isShortSet {
		inferredInstructions = append(inferredInstructions, newSignalInstruction("LONG", 0, true))
	}
	if sto.SignalInput.InvalidateAfterSeconds == 0 {
		inferredInstructions = append(inferredInstructions, newSignalInstruction("INVALIDATE AFTER 2 DAYS", 0, true))
	}
	if sto.SignalInput.EnterRangeHigh == 0 && sto.SignalInput.EnterRangeLow == 0 {
		inferredInstructions = append(inferredInstructions, newSignalInstruction("ENTER: IMMEDIATELY", 0, true))
	}
	return inferredInstructions
}

func (t SignalTranspiler) calculateErrorsAndWarnings(sto *SignalTranspilerOutput) {
	if sto.SignalInput.BaseAsset == "" || sto.SignalInput.QuoteAsset == "" {
		sto.Errors = append(sto.Errors, fmt.Errorf("%w, e.g. MARKET: BTC/USDT", errMarketRequired).Error())
	}
	if sto.SignalInput.InitialISO8601 == "" {
		sto.Errors = append(sto.Errors, fmt.Errorf("%w, e.g. START AT: 2021-06-22T15:21:03Z", errInitialISO8601Required).Error())
	}
	if sto.SignalInput.EnterRangeLow == 0.0 && sto.SignalInput.EnterRangeHigh == 0.0 {
		sto.Errors = append(sto.Errors, fmt.Errorf("%w, e.g. ENTER BETWEEN: 0.1 - 0.5 or ENTER: IMMEDIATELY", errEnterRangeRequired).Error())
	}
}

type inputToken struct {
	Input     string `json:"input"`
	TokenType string `json:"tokenType"`
}

type signalInstruction struct {
	rawInput       string
	lineNumber     int
	err            error
	tokenizedInput []inputToken
	isApplied      bool
	isInferred     bool
}

func newSignalInstruction(rawInput string, lineNumber int, isInferred bool) *signalInstruction {
	return &signalInstruction{rawInput: rawInput, lineNumber: lineNumber, isInferred: isInferred}
}

func (si *signalInstruction) apply(input SignalTranspilerOutput) (SignalTranspilerOutput, error) {
	if si.isApplied {
		return input, nil
	}
	for _, instruction := range instructions {
		resultInstruction, ok := instruction.apply(si.rawInput, &input)
		if ok {
			si.tokenizedInput = resultInstruction.tokenizedInput
			if si.isInferred {
				si.tokenizedInput = append(si.tokenizedInput, inputToken{Input: " // INFERRED", TokenType: TOKEN_COMMENT})
			}
			si.err = resultInstruction.err
			si.isApplied = true
			return input, si.err
		}
	}
	err := fmt.Errorf("%w at line %v with content [%v]", errUnrecognizedInstruction, si.lineNumber, si.rawInput)
	si.err = err
	si.isApplied = true
	return input, err
}

const (
	TOKEN_INSTRUCTION = "instruction"
	TOKEN_PUNCTUATION = "punctuation"
	TOKEN_EXPRESSION  = "expression"
	TOKEN_COMMENT     = "comment"
	TOKEN_ERROR       = "error"
)

var (
	errUnrecognizedInstruction            = errors.New("unrecognized instruction")
	errMarketAlreadySupplied              = errors.New("market already supplied")
	errEnterRangeAlreadySupplied          = errors.New("enter range already supplied")
	errInvalidateAfterDaysAlreadySupplied = errors.New("timeout after days already supplied")
	errIsShortAlreadySupplied             = errors.New("short/long already supplied")
	errInitialISO8601AlreadySupplied      = errors.New("'start at' already supplied")
	errExchangeAlreadySupplied            = errors.New("exchange already supplied")
	errStopLossAlreadySupplied            = errors.New("stop loss already supplied")
	errMaximumInvalidation7Days           = errors.New("maximum timeout after 7 days")
	errMalformedInteger                   = errors.New("malformed integer")
	errMalformedFloat                     = errors.New("malformed float")
	errInvalidEnterRange                  = errors.New("invalid enter range")
	errInvalidEnterAt                     = errors.New("invalid 'enter at' format")
	errUnsupportedExchange                = errors.New("unsupported exchange")
	errUnsupportedDateTimeFormat          = errors.New("unsupported datetime format")
	errMarketRequired                     = errors.New("'market' required")
	errEnterRangeRequired                 = errors.New("enter range required")
	errInitialISO8601Required             = errors.New("'start at' required")
	errMixesSeparators                    = errors.New("mixing number separators is not supported, use comma, dash or AND")
)

type instruction interface {
	apply(rawInput string, sto *SignalTranspilerOutput) (signalInstruction, bool)
}
