package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	tasksora "github.com/QuantumNous/new-api/relay/channel/task/sora"
	"github.com/stretchr/testify/require"
)

func TestNewAPIChannelUsesOpenAIVideoTaskAdaptor(t *testing.T) {
	platform := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI))
	require.IsType(t, &tasksora.TaskAdaptor{}, GetTaskAdaptor(platform))
}
