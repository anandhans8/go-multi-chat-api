package signal_client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	logger "go-multi-chat-api/src/infrastructure/logger"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	uuid "github.com/gofrs/uuid"
	//log "github.com/sirupsen/logrus"
	"go-multi-chat-api/src/infrastructure/utils"

	"github.com/tidwall/sjson"
)

type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type JsonRpc2MessageResponse struct {
	Id     string          `json:"id"`
	Result json.RawMessage `json:"result"`
	Err    Error           `json:"error"`
}

type JsonRpc2ReceivedMessage struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Err    Error           `json:"error"`
}

type RateLimitMessage struct {
	Response RateLimitResponse `json:"response"`
}

type RateLimitResponse struct {
	Results []RateLimitResult `json:"results"`
}

type RateLimitResult struct {
	Token string `json:"token"`
}

type RateLimitErrorType struct {
	ChallengeTokens []string
	Err             error
}

func (r *RateLimitErrorType) Error() string {
	return r.Err.Error()
}

type JsonRpc2Client struct {
	conn                     net.Conn
	receivedResponsesById    map[string]chan JsonRpc2MessageResponse
	receivedMessagesChannels map[string]chan JsonRpc2ReceivedMessage
	lastTimeErrorMessageSent time.Time
	signalCliApiConfig       *utils.SignalCliApiConfig
	number                   string
	receivedMessagesMutex    sync.Mutex
	receivedResponsesMutex   sync.Mutex
	Logger                   *logger.Logger
}

func NewJsonRpc2Client(signalCliApiConfig *utils.SignalCliApiConfig, number string, loggerInstance *logger.Logger) *JsonRpc2Client {
	return &JsonRpc2Client{
		signalCliApiConfig:       signalCliApiConfig,
		number:                   number,
		receivedResponsesById:    make(map[string]chan JsonRpc2MessageResponse),
		receivedMessagesChannels: make(map[string]chan JsonRpc2ReceivedMessage),
		Logger:                   loggerInstance,
	}
}

func (r *JsonRpc2Client) Dial(address string) error {
	var err error
	r.conn, err = net.Dial("tcp", address)
	if err != nil {
		return err
	}

	return nil
}

func (r *JsonRpc2Client) getRaw(command string, account *string, args interface{}) (string, error) {
	type Request struct {
		JsonRpc string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Id      string      `json:"id"`
		Params  interface{} `json:"params,omitempty"`
	}

	trustModeStr := ""
	trustMode, err := r.signalCliApiConfig.GetTrustModeForNumber(r.number)
	if err == nil {
		trustModeStr, err = utils.TrustModeToString(trustMode)
		if err != nil {
			trustModeStr = ""
			r.Logger.Error(fmt.Sprintf("Invalid trust mode: %s", trustModeStr))
		}
	}

	u, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	fullCommand := Request{JsonRpc: "2.0", Method: command, Id: u.String()}
	if args != nil {
		fullCommand.Params = args
	}

	fullCommandBytes, err := json.Marshal(fullCommand)
	if err != nil {
		return "", err
	}

	if trustModeStr != "" {
		fullCommandBytes, err = sjson.SetBytes(fullCommandBytes, "params.trustNewIdentities", trustModeStr)
		if err != nil {
			return "", err
		}
	}

	if account != nil {
		fullCommandBytes, err = sjson.SetBytes(fullCommandBytes, "params.account", account)
		if err != nil {
			return "", err
		}
	}

	r.Logger.Debug(fmt.Sprintf("json-rpc command: %s", string(fullCommandBytes)))

	_, err = r.conn.Write([]byte(string(fullCommandBytes) + "\n"))
	if err != nil {
		return "", err
	}

	responseChan := make(chan JsonRpc2MessageResponse)
	r.receivedResponsesMutex.Lock()
	r.receivedResponsesById[u.String()] = responseChan
	r.receivedResponsesMutex.Unlock()

	var resp JsonRpc2MessageResponse
	resp = <-responseChan

	r.receivedResponsesMutex.Lock()
	delete(r.receivedResponsesById, u.String())
	r.receivedResponsesMutex.Unlock()

	r.Logger.Debug(fmt.Sprintf("json-rpc command response: %s", string(resp.Result)))
	r.Logger.Debug(fmt.Sprintf("json-rpc response error: %s", resp.Err.Message))

	if resp.Err.Code != 0 {
		r.Logger.Debug(fmt.Sprintf("json-rpc command error code: %d", resp.Err.Code))
		if resp.Err.Code == -5 {
			var rateLimitMessage RateLimitMessage
			err = json.Unmarshal(resp.Err.Data, &rateLimitMessage)
			if err != nil {
				return "", errors.New(resp.Err.Message + " (Couldn't parse JSON for more details")
			}
			challengeTokens := []string{}
			for _, rateLimitResult := range rateLimitMessage.Response.Results {
				challengeTokens = append(challengeTokens, rateLimitResult.Token)
			}

			return "", &RateLimitErrorType{
				ChallengeTokens: challengeTokens,
				Err:             errors.New(resp.Err.Message),
			}
		}
		return "", errors.New(resp.Err.Message)
	}

	return string(resp.Result), nil
}

func postMessageToWebhook(webhookUrl string, data []byte) error {
	r, err := http.NewRequest("POST", webhookUrl, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	r.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 && res.StatusCode != 201 {
		return errors.New("Unexpected status code returned (" + strconv.Itoa(res.StatusCode) + ")")
	}
	return nil
}

func (r *JsonRpc2Client) ReceiveData(number string, receiveWebhookUrl string) {
	connbuf := bufio.NewReader(r.conn)
	for {
		str, err := connbuf.ReadString('\n')
		if err != nil {
			elapsed := time.Since(r.lastTimeErrorMessageSent)
			if (elapsed) > time.Duration(5*time.Minute) { //avoid spamming the log file and only log the message at max every 5 minutes
				r.Logger.Error(fmt.Sprintf("Couldn't read data for number %s: %s . Is the number properly registered?", number, err.Error()))

				r.lastTimeErrorMessageSent = time.Now()
			}
			continue
		}
		r.Logger.Debug(fmt.Sprintf("json-rpc received data: %s", str))

		if receiveWebhookUrl != "" {
			err = postMessageToWebhook(receiveWebhookUrl, []byte(str))
			if err != nil {
				r.Logger.Error("Couldn't post data to webhook:", zap.Error(err))
			}
		}

		var resp1 JsonRpc2ReceivedMessage
		json.Unmarshal([]byte(str), &resp1)
		if resp1.Method == "receive" {
			r.receivedMessagesMutex.Lock()
			for _, c := range r.receivedMessagesChannels {
				select {
				case c <- resp1:
					r.Logger.Debug("Message sent to golang channel")
				default:
					r.Logger.Debug("Couldn't send message to golang channel, as there's no receiver")
				}
				continue
			}
			r.receivedMessagesMutex.Unlock()
		}

		var resp2 JsonRpc2MessageResponse
		err = json.Unmarshal([]byte(str), &resp2)
		if err == nil {
			if resp2.Id != "" {
				if responseChan, ok := r.receivedResponsesById[resp2.Id]; ok {
					responseChan <- resp2
				}
			}
		} else {
			r.Logger.Error(fmt.Sprintf("Received unparsable message: %s", str))
		}
	}
}

func (r *JsonRpc2Client) GetReceiveChannel() (chan JsonRpc2ReceivedMessage, string, error) {
	c := make(chan JsonRpc2ReceivedMessage)

	channelUuid, err := uuid.NewV4()
	if err != nil {
		return c, "", err
	}

	r.receivedMessagesMutex.Lock()
	r.receivedMessagesChannels[channelUuid.String()] = c
	r.receivedMessagesMutex.Unlock()

	return c, channelUuid.String(), nil
}

func (r *JsonRpc2Client) RemoveReceiveChannel(channelUuid string) {
	r.receivedMessagesMutex.Lock()
	delete(r.receivedMessagesChannels, channelUuid)
	r.receivedMessagesMutex.Unlock()
}
