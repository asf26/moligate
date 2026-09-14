package openai

import (
	"fmt"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func OpenaiRealtimeHandler(c *gin.Context, info *relaycommon.RelayInfo) (*types.NewAPIError, *dto.RealtimeUsage) {
	if info == nil || info.ClientWs == nil || info.TargetWs == nil {
		return types.NewError(fmt.Errorf("invalid websocket connection"), types.ErrorCodeBadResponse), nil
	}

	info.IsStream = true
	clientConn := info.ClientWs
	targetConn := info.TargetWs

	clientClosed := make(chan struct{})
	targetClosed := make(chan struct{})
	sendChan := make(chan []byte, 100)
	receiveChan := make(chan []byte, 100)
	errChan := make(chan error, 2)

	var usageMu sync.Mutex
	localUsage := dto.RealtimeUsage{}
	sumUsage := dto.RealtimeUsage{}

	gopool.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in client reader: %v", r)
			}
		}()
		for {
			select {
			case <-c.Done():
				return
			default:
				_, message, err := clientConn.ReadMessage()
				if err != nil {
					if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						close(clientClosed)
					} else {
						errChan <- fmt.Errorf("error reading from client: %w", err)
					}
					return
				}

				realtimeEvent := &dto.RealtimeEvent{}
				err = common.Unmarshal(message, realtimeEvent)
				if err != nil {
					errChan <- fmt.Errorf("error unmarshalling message: %v", err)
					return
				}

				if realtimeEvent.Type == dto.RealtimeEventTypeSessionUpdate {
					if realtimeEvent.Session != nil {
						if realtimeEvent.Session.Tools != nil {
							info.RealtimeTools = realtimeEvent.Session.Tools
						}
					}
				}

				textToken, audioToken, err := service.CountTokenRealtime(info, *realtimeEvent, info.UpstreamModelName)
				if err != nil {
					errChan <- fmt.Errorf("error counting text token: %v", err)
					return
				}
				logger.LogInfo(c, fmt.Sprintf("type: %s, textToken: %d, audioToken: %d", realtimeEvent.Type, textToken, audioToken))
				eventUsage := &dto.RealtimeUsage{
					TotalTokens: textToken + audioToken,
					InputTokens: textToken + audioToken,
					InputTokenDetails: dto.InputTokenDetails{
						TextTokens:  textToken,
						AudioTokens: audioToken,
					},
				}
				usageMu.Lock()
				err = reservePendingUsage(c, info, &sumUsage, &localUsage, eventUsage)
				usageMu.Unlock()
				if err != nil {
					errChan <- fmt.Errorf("error reserving input usage: %w", err)
					return
				}

				err = helper.WssString(c, targetConn, string(message))
				if err != nil {
					errChan <- fmt.Errorf("error writing to target: %v", err)
					return
				}

				select {
				case sendChan <- message:
				default:
				}
			}
		}
	})

	gopool.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in target reader: %v", r)
			}
		}()
		for {
			select {
			case <-c.Done():
				return
			default:
				_, message, err := targetConn.ReadMessage()
				if err != nil {
					if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						close(targetClosed)
					} else {
						errChan <- fmt.Errorf("error reading from target: %w", err)
					}
					return
				}
				info.SetFirstResponseTime()
				realtimeEvent := &dto.RealtimeEvent{}
				err = common.Unmarshal(message, realtimeEvent)
				if err != nil {
					errChan <- fmt.Errorf("error unmarshalling message: %v", err)
					return
				}

				if realtimeEvent.Type == dto.RealtimeEventTypeResponseDone {
					var realtimeUsage *dto.RealtimeUsage
					if realtimeEvent.Response != nil {
						realtimeUsage = realtimeEvent.Response.Usage
					}
					if realtimeUsage != nil {
						usageMu.Lock()
						err := preConsumeUsage(c, info, realtimeUsage, &sumUsage)
						if err == nil {
							localUsage = dto.RealtimeUsage{}
						}
						usageMu.Unlock()
						if err != nil {
							errChan <- fmt.Errorf("error consume usage: %w", err)
							return
						}
					} else {
						textToken, audioToken, err := service.CountTokenRealtime(info, *realtimeEvent, info.UpstreamModelName)
						if err != nil {
							errChan <- fmt.Errorf("error counting text token: %v", err)
							return
						}
						logger.LogInfo(c, fmt.Sprintf("type: %s, textToken: %d, audioToken: %d", realtimeEvent.Type, textToken, audioToken))
						info.IsFirstRequest = false
						eventUsage := &dto.RealtimeUsage{
							TotalTokens: textToken + audioToken,
							InputTokens: textToken + audioToken,
							InputTokenDetails: dto.InputTokenDetails{
								TextTokens:  textToken,
								AudioTokens: audioToken,
							},
						}
						usageMu.Lock()
						err = reservePendingUsage(c, info, &sumUsage, &localUsage, eventUsage)
						if err == nil {
							addRealtimeUsage(&sumUsage, &localUsage)
							localUsage = dto.RealtimeUsage{}
						}
						usageMu.Unlock()
						if err != nil {
							errChan <- fmt.Errorf("error consume usage: %w", err)
							return
						}
					}
					usageMu.Lock()
					logger.LogInfo(c, fmt.Sprintf("realtime streaming sumUsage: %v", sumUsage))
					logger.LogInfo(c, fmt.Sprintf("realtime streaming localUsage: %v", localUsage))
					usageMu.Unlock()

				} else if realtimeEvent.Type == dto.RealtimeEventTypeSessionUpdated || realtimeEvent.Type == dto.RealtimeEventTypeSessionCreated {
					realtimeSession := realtimeEvent.Session
					if realtimeSession != nil {
						// update audio format
						info.InputAudioFormat = common.GetStringIfEmpty(realtimeSession.InputAudioFormat, info.InputAudioFormat)
						info.OutputAudioFormat = common.GetStringIfEmpty(realtimeSession.OutputAudioFormat, info.OutputAudioFormat)
					}
				} else {
					textToken, audioToken, err := service.CountTokenRealtime(info, *realtimeEvent, info.UpstreamModelName)
					if err != nil {
						errChan <- fmt.Errorf("error counting text token: %v", err)
						return
					}
					logger.LogInfo(c, fmt.Sprintf("type: %s, textToken: %d, audioToken: %d", realtimeEvent.Type, textToken, audioToken))
					eventUsage := &dto.RealtimeUsage{
						TotalTokens:  textToken + audioToken,
						OutputTokens: textToken + audioToken,
						OutputTokenDetails: dto.OutputTokenDetails{
							TextTokens:  textToken,
							AudioTokens: audioToken,
						},
					}
					usageMu.Lock()
					err = reservePendingUsage(c, info, &sumUsage, &localUsage, eventUsage)
					usageMu.Unlock()
					if err != nil {
						errChan <- fmt.Errorf("error reserving output usage: %w", err)
						return
					}
				}

				err = helper.WssString(c, clientConn, string(message))
				if err != nil {
					errChan <- fmt.Errorf("error writing to client: %v", err)
					return
				}

				select {
				case receiveChan <- message:
				default:
				}
			}
		}
	})

	var handlerErr error
	select {
	case <-clientClosed:
	case <-targetClosed:
	case err := <-errChan:
		logger.LogError(c, "realtime error: "+err.Error())
		handlerErr = err
	case <-c.Done():
	}

	usageMu.Lock()
	// Every pending event was reserved before it was forwarded. Promote that
	// reserved prefix for final settlement even when response.done is absent or
	// its authoritative total exceeds the remaining allowance.
	if localUsage.TotalTokens != 0 {
		addRealtimeUsage(&sumUsage, &localUsage)
		localUsage = dto.RealtimeUsage{}
	}
	finalUsage := sumUsage
	usageMu.Unlock()

	if handlerErr != nil {
		// Return the successfully reserved prefix as well. WssHelper settles it
		// before propagating the error, so delivered events are never refunded.
		if apiErr, ok := handlerErr.(*types.NewAPIError); ok {
			return apiErr, &finalUsage
		}
		return types.NewError(handlerErr, types.ErrorCodeBadResponse, types.ErrOptionWithSkipRetry()), &finalUsage
	}

	return nil, &finalUsage
}

func addRealtimeUsage(totalUsage *dto.RealtimeUsage, usage *dto.RealtimeUsage) {
	totalUsage.TotalTokens += usage.TotalTokens
	totalUsage.InputTokens += usage.InputTokens
	totalUsage.OutputTokens += usage.OutputTokens
	totalUsage.InputTokenDetails.CachedTokens += usage.InputTokenDetails.CachedTokens
	totalUsage.InputTokenDetails.TextTokens += usage.InputTokenDetails.TextTokens
	totalUsage.InputTokenDetails.AudioTokens += usage.InputTokenDetails.AudioTokens
	totalUsage.OutputTokenDetails.TextTokens += usage.OutputTokenDetails.TextTokens
	totalUsage.OutputTokenDetails.AudioTokens += usage.OutputTokenDetails.AudioTokens
}

// reservePendingUsage grows the request reservation before an event is
// forwarded. pendingUsage remains unmodified when the reservation is rejected.
func reservePendingUsage(ctx *gin.Context, info *relaycommon.RelayInfo, committedUsage, pendingUsage, eventUsage *dto.RealtimeUsage) error {
	if committedUsage == nil || pendingUsage == nil || eventUsage == nil {
		return fmt.Errorf("invalid usage pointer")
	}
	nextPending := *pendingUsage
	addRealtimeUsage(&nextPending, eventUsage)
	candidate := *committedUsage
	addRealtimeUsage(&candidate, &nextPending)
	if err := service.ReserveWssConsumeQuota(ctx, info, &candidate); err != nil {
		return err
	}
	*pendingUsage = nextPending
	return nil
}

func preConsumeUsage(ctx *gin.Context, info *relaycommon.RelayInfo, usage *dto.RealtimeUsage, totalUsage *dto.RealtimeUsage) error {
	if usage == nil || totalUsage == nil {
		return fmt.Errorf("invalid usage pointer")
	}

	candidate := *totalUsage
	addRealtimeUsage(&candidate, usage)
	// Grow one request-scoped reservation from cumulative usage. The final
	// websocket settlement uses this same cumulative value, so individual
	// response events are not charged a second time.
	if err := service.ReserveWssConsumeQuota(ctx, info, &candidate); err != nil {
		return err
	}
	*totalUsage = candidate
	return nil
}
