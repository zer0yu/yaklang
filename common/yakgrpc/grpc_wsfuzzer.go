package yakgrpc

import (
	"bytes"
	"encoding/json"
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/lowhttp"
	"github.com/yaklang/yaklang/common/yakgrpc/ypb"
	"strconv"
	"strings"
	"sync"
)

func (s *Server) CreateWebsocketFuzzer(stream ypb.Yak_CreateWebsocketFuzzerServer) error {
	firstReq, err := stream.Recv()
	if err != nil {
		return utils.Errorf("first websocket fuzzer: %s", err)
	}
	_ = firstReq

	var id int64 = 0
	var _requireIndexLock = new(sync.Mutex)
	requireDataFrameID := func() int64 {
		_requireIndexLock.Lock()
		defer _requireIndexLock.Unlock()
		id++
		return id
	}

	client, err := lowhttp.NewWebsocketClient(
		firstReq.GetUpgradeRequest(),
		lowhttp.WithWebsocketWithContext(stream.Context()),
		lowhttp.WithWebsocketTLS(firstReq.GetIsTLS()),
		lowhttp.WithWebsocketTotalTimeout(float64(firstReq.GetTotalTimeoutSeconds())),
		lowhttp.WithWebsocketProxy(firstReq.GetProxy()),
		lowhttp.WithWebsocketFromServerHandler(func(fromServer []byte) {
			var encoded []string
			if utils.IsGzip(fromServer) {
				encoded = append(encoded, "gzip")
			}

			_, isJson := utils.IsJSON(string(fromServer))
			var dataVerbose = utils.EscapeInvalidUTF8Byte(fromServer)
			if isJson {
				var buf bytes.Buffer
				_ = json.Indent(&buf, fromServer, "", "")
				dataVerbose = strings.ReplaceAll(string(buf.Bytes()), "\n", "")
				var formattedBuf bytes.Buffer
				_ = json.Indent(&formattedBuf, fromServer, "", "    ")
				if formattedBuf.Len() > 0 {
					fromServer = formattedBuf.Bytes()
				}
			}
			if dataVerbose == "" {
				dataVerbose = strings.Trim(strconv.Quote(string(fromServer)), `"`)
			}

			size := len(fromServer)
			msg := &ypb.ClientWebsocketResponse{
				SwitchProtocolSucceeded: true,
				IsDataFrame:             true,
				FromServer:              true,
				Data:                    fromServer,
				DataVerbose:             dataVerbose,
				DataLength:              int64(size),
				DataSizeVerbose:         utils.ByteSize(uint64(size)),
				IsJson:                  isJson,
				IsProtobuf:              utils.IsProtobuf(fromServer),
				DataFrameIndex:          requireDataFrameID(),
			}
			stream.Send(msg)
		}),
	)
	if err != nil {
		return utils.Errorf("websocket client build tunnel failed: %s", err)
	}

	stream.Send(&ypb.ClientWebsocketResponse{
		IsUpgradeResponse: true,
		UpgradeResponse:   client.Response,
	})

	client.StartFromServer()

	go func() {
		defer func() {
			client.Stop()
		}()

		for {
			req, err := stream.Recv()
			if err != nil {
				log.Errorf("stream recv wsfuzzer req failed: %s", err)
				return
			}
			toServerStr := string(req.GetToServer())
			client.Write(req.GetToServer())

			_, isJson := utils.IsJSON(string(req.GetToServer()))
			var dataVerbose = ""
			if isJson {
				var buf bytes.Buffer
				_ = json.Indent(&buf, req.GetToServer(), "", "")
				if buf.Len() > 0 {
					dataVerbose = strings.ReplaceAll(string(buf.Bytes()), "\n", "")
				}
			}
			if dataVerbose == "" {
				dataVerbose = strings.Trim(strconv.Quote(toServerStr), `"`)
			}

			msg := &ypb.ClientWebsocketResponse{
				SwitchProtocolSucceeded: true,
				IsDataFrame:             true,
				FromServer:              false,
				Data:                    []byte(toServerStr),
				DataVerbose:             dataVerbose,
				DataLength:              int64(len(toServerStr)),
				DataSizeVerbose:         utils.ByteSize(uint64(len(toServerStr))),
				IsJson:                  isJson,
				IsProtobuf:              utils.IsProtobuf([]byte(toServerStr)),
				DataFrameIndex:          requireDataFrameID(),
			}
			stream.Send(msg)
		}
	}()
	client.Wait()
	return utils.Errorf("unexpected error: %s", "unknown")
}
