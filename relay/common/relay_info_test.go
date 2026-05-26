package common

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoAppendRequestConversionMeta(t *testing.T) {
	info := &RelayInfo{}

	info.AppendRequestConversionMeta(" assistant_prefill_continuation ")
	info.AppendRequestConversionMeta("assistant_prefill_continuation")
	info.AppendRequestConversionMeta("")

	require.Equal(t, []string{"assistant_prefill_continuation"}, info.RequestConversionMeta)
}

func TestRelayInfoResetAttemptConversionMeta(t *testing.T) {
	info := &RelayInfo{
		RequestConversionMeta: []string{"assistant_prefill_continuation"},
	}
	info.SetEstimatePromptTokens(123)

	info.ResetAttemptConversionMeta(99)

	require.Empty(t, info.RequestConversionMeta)
	require.Equal(t, 99, info.GetEstimatePromptTokens())
}
