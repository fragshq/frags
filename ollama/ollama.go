package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fragshq/frags"
	"github.com/fragshq/frags/log"
	"github.com/fragshq/frags/resources"
	"github.com/fragshq/frags/schema"
	"github.com/fragshq/frags/util"
)

const engine = "ollama"
const defaultModel = "qwen3:latest"
const temperature float32 = 0.1
const topK float32 = 40
const topP float32 = 0.9

// Ai implements the Ai interface for Ollama.
type Ai struct {
	client       *http.Client
	baseURL      string
	config       Config
	messages     []Message
	systemPrompt string
	Functions    frags.ExternalFunctions
}

type Config struct {
	Model       string  `yaml:"model" json:"model"`
	Temperature float32 `yaml:"temperature" json:"temperature"`
	TopK        float32 `yaml:"topK" json:"topK"`
	TopP        float32 `yaml:"topP" json:"topP"`
	NumPredict  int     `yaml:"numPredict" json:"numPredict"`
}

func DefaultConfig() Config {
	return Config{
		Model:       defaultModel,
		Temperature: temperature,
		TopK:        topK,
		TopP:        topP,
		NumPredict:  1024,
	}
}

// NewAI creates a new Ai instance
func NewAI(baseURL string, config Config) *Ai {
	return &Ai{
		baseURL: baseURL,
		config:  config,
		client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 10,
			},
		},
		Functions: frags.ExternalFunctions{},
		messages:  make([]Message, 0),
	}
}

// New creates a new Ai instance and returns it
func (d *Ai) New() frags.Ai {
	return &Ai{
		client:       d.client,
		baseURL:      d.baseURL,
		config:       d.config,
		messages:     make([]Message, 0),
		Functions:    d.Functions,
		systemPrompt: d.systemPrompt,
	}
}

// Ask performs a query against the Ollama API, according to the Frags interface
func (d *Ai) Ask(ctx *util.FragsContext, text string, sx *schema.Schema, tools frags.ToolDefinitions,
	runner frags.ExportableRunner, rx ...resources.ResourceData) ([]byte, error) {
	if len(text) == 0 {
		if sx == nil && len(d.messages) > 0 {
			return []byte(d.messages[len(d.messages)-1].Content), nil
		}
		return nil, fmt.Errorf("no prompt provided but was required")
	}
	if len(d.systemPrompt) > 0 && len(d.messages) == 0 {
		d.messages = append(d.messages, Message{
			Content: d.systemPrompt,
			Role:    "system",
		})
	}
	message := Message{
		Content: "",
		Role:    "user",
	}
	for _, r := range rx {
		if r.MediaType != util.MediaText {
			return nil, errors.New("ollama only supports text resources")
		}
		message.Content += message.Content + " === " + r.Identifier + " === \n" + string(r.ByteContent) + "\n===\n"
		runner.Logger().Debug(log.NewEvent(log.LoadEventType, log.AiComponent).WithMessage("adding file resource").WithEngine(engine).WithResource(r.Identifier))
	}
	useThinking := strings.HasPrefix(text, "/think")
	message.Content += "\n" + text
	d.messages = append(d.messages, message)
	request := Request{
		Messages: d.messages,
		Model:    d.config.Model,
		Think:    useThinking,
		Format:   sx,
		Tools:    make([]ToolDefinition, 0),
		Options: Options{
			NumPredict:  d.config.NumPredict,
			Temperature: d.config.Temperature,
			TopK:        d.config.TopK,
			TopP:        d.config.TopP,
		},
	}
	var resolved frags.ResolvedTools
	var err error
	request.Tools, resolved, err = d.configureTools(tools)
	if err != nil {
		return nil, err
	}
	runner.Logger().Debug(log.NewEvent(log.GenericEventType, log.AiComponent).WithMessage("configured tools").WithEngine(engine).WithContent(request.Tools))
	keepGoing := true
	out := ""
	for keepGoing {
		request.Messages = d.messages
		err := func() error {
			runner.Logger().Debug(log.NewEvent(log.StartEventType, log.AiComponent).WithMessage("generating content").WithEngine(engine).WithContent(request.Messages[len(request.Messages)-1]))
			responseMessage, err := d.sendRequest(ctx, request)
			if err != nil {
				return err
			}
			d.messages = append(d.messages, responseMessage.Message)
			if len(responseMessage.Message.ToolCalls) > 0 {
				err := d.handleFunctionCall(ctx, responseMessage, resolved, runner)
				if err != nil {
					return err
				}
			} else {
				out = responseMessage.Message.Content
				runner.Logger().Debug(log.NewEvent(log.EndEventType, log.AiComponent).WithMessage("generated content").WithEngine(engine).WithContent(out))
				keepGoing = false
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return []byte(out), nil
}

func (d *Ai) handleFunctionCall(ctx *util.FragsContext, responseMessage Response, resolved frags.ResolvedTools, runner frags.ExportableRunner) error {
	for _, fc := range responseMessage.Message.ToolCalls {
		res, err := resolved.Run(ctx, fc.Function.Name, fc.Function.Arguments, runner)
		if err != nil {
			return err
		}
		content, err := json.Marshal(res)
		if err != nil {
			return err
		}
		d.messages = append(d.messages, Message{
			Role:       "tool",
			Content:    string(content),
			ToolCallID: fc.ID,
		})
	}
	return nil
}

func (d *Ai) SetSystemPrompt(systemPrompt string) {
	d.systemPrompt = systemPrompt
}

func (d *Ai) SetFunctions(functions frags.ExternalFunctions) {
	d.Functions = functions
}

func (d *Ai) sendRequest(ctx context.Context, request Request) (Response, error) {
	requestBody, err := json.Marshal(request)
	response := Response{}
	if err != nil {
		return response, err
	}
	reader := bytes.NewReader(requestBody)
	apiUrl, err := url.Parse(d.baseURL + "/api/chat")
	if err != nil {
		return response, err
	}
	req := http.Request{
		Method: "POST",
		URL:    apiUrl,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(reader),
	}

	res, err := d.client.Do(req.WithContext(ctx))
	if err != nil {
		return response, err
	}
	defer func() {
		if res.Body != nil {
			_ = res.Body.Close()
		}
	}()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(responseBody, &response)
	return response, err
}

func (d *Ai) configureTools(tools frags.ToolDefinitions) ([]ToolDefinition, frags.ResolvedTools, error) {
	resolved, err := tools.Resolve(d.Functions)
	if err != nil {
		return nil, nil, err
	}
	tx := make([]ToolDefinition, 0)
	for _, rt := range resolved {
		// Parameters has no omitempty, so a nil schema would serialize as "parameters": null.
		// Tools without arguments get an explicit object schema instead
		parameters := rt.Schema
		if parameters == nil {
			parameters = &schema.Schema{Type: schema.Object}
		}
		tx = append(tx, ToolDefinition{
			Type: "function",
			Function: FunctionDef{
				Name:        rt.ExposedName,
				Description: rt.Description,
				Parameters:  parameters,
			},
		})
	}
	return tx, resolved, nil
}

func (d *Ai) RunFunction(ctx *util.FragsContext, functionCall frags.FunctionCaller, runner frags.ExportableRunner) (any, error) {
	return runner.RunFunction(ctx, functionCall.Name, functionCall.Args)
}
