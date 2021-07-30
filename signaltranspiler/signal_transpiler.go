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
	Errors         []string                `json:"errors"`
	Warnings       []string                `json:"warnings"`
	TokenizedInput [][]inputToken          `json:"tokenizedInput"`
	SignalInput    common.SignalCheckInput `json:"signalInput"`
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
		signalInstructions = append(signalInstructions, newSignalInstruction(line, i))
	}
	output := SignalTranspilerOutput{
		SignalInput: common.SignalCheckInput{},
		Errors:      []string{},
		Warnings:    []string{},
	}
	// 1. Instructions may be deferred, so do passes until the number of transpiled instructions is 0
	// 2. Do an inference pass (i.e. apply defaults)
	// 3. Do passes again until the number of transpiled instructions is 0
	// 4. Do a validation pass: if required params are missing, fail. If there are still untranspiled instructions, fail.
	for _, signalInstruction := range signalInstructions {
		var err error
		output.SignalInput, err = signalInstruction.apply(output.SignalInput)
		if err != nil {
			output.Errors = append(output.Errors, err.Error())
			continue
		}
	}

	for _, signalInstruction := range signalInstructions {
		if len(signalInstruction.tokenizedInput) == 0 {
			output.TokenizedInput = append(output.TokenizedInput, []inputToken{{Input: signalInstruction.rawInput, TokenType: TOKEN_ERROR}})
			continue
		}
		output.TokenizedInput = append(output.TokenizedInput, signalInstruction.tokenizedInput)
	}

	return output, output.error()
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
}

func newSignalInstruction(rawInput string, lineNumber int) *signalInstruction {
	return &signalInstruction{rawInput: rawInput, lineNumber: lineNumber}
}

func (si *signalInstruction) apply(input common.SignalCheckInput) (common.SignalCheckInput, error) {
	for _, instruction := range instructions {
		resultInstruction, ok := instruction.apply(si.rawInput, &input)
		if ok {
			si.tokenizedInput = resultInstruction.tokenizedInput
			si.err = resultInstruction.err
			return input, si.err
		}
	}
	err := fmt.Errorf("%w at line %v with content [%v]", errUnrecognizedInstruction, si.lineNumber, si.rawInput)
	si.err = err
	return common.SignalCheckInput{}, err
}

const (
	TOKEN_INSTRUCTION = "instruction"
	TOKEN_PUNCTUATION = "punctuation"
	TOKEN_EXPRESSION  = "expression"
	TOKEN_ERROR       = "error"
)

var (
	errUnrecognizedInstruction = errors.New("unrecognized instruction")
	errMalformedFloat          = errors.New("malformed float")
)

type instruction interface {
	apply(rawInput string, signalInput *common.SignalCheckInput) (signalInstruction, bool)
}
