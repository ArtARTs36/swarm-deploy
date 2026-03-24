package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/artarts36/swarm-deploy/internal/assistant/conversation"
	"github.com/artarts36/swarm-deploy/internal/assistant/guard"
	"github.com/artarts36/swarm-deploy/internal/assistant/rag"
	"github.com/artarts36/swarm-deploy/internal/service"
	"github.com/tmc/langchaingo/llms"
	langgraph "github.com/tmc/langgraphgo/graph"
)

const maxToolIterations = 3

const (
	graphNodeGuard          = "guard"
	graphNodeRetrieve       = "retrieve_context"
	graphNodePrepare        = "prepare_messages"
	graphNodeGenerateAnswer = "generate_answer"
)

var (
	errPromptInjection = errors.New("request rejected by prompt injection guard")
)

type graph struct {
	config         Config
	guard          *guard.InjectionChecker
	retriever      *rag.Retriever
	chat           *openAIClient
	tools          ToolExecutor
	allowedToolSet map[string]struct{}
}

type graphExecutionState struct {
	history            []conversation.Turn
	userMessage        string
	relevantServices   []service.Info
	modelMessages      []modelMessage
	collectedToolCalls []ToolCall
	answer             string
}

func newGraph(
	config Config,
	guard *guard.InjectionChecker,
	retriever *rag.Retriever,
	chat *openAIClient,
	tools ToolExecutor,
	allowedToolSet map[string]struct{},
) *graph {
	return &graph{
		config:         config,
		guard:          guard,
		retriever:      retriever,
		chat:           chat,
		tools:          tools,
		allowedToolSet: allowedToolSet,
	}
}

func (g *graph) run(ctx context.Context, history []conversation.Turn, userMessage string) (string, []ToolCall, error) {
	executionState := &graphExecutionState{
		history:            history,
		userMessage:        userMessage,
		collectedToolCalls: make([]ToolCall, 0, 2),
	}

	runnable, err := g.compile(executionState)
	if err != nil {
		return "", executionState.collectedToolCalls, err
	}

	if _, err := runnable.Invoke(ctx, nil); err != nil {
		return "", executionState.collectedToolCalls, err
	}

	return executionState.answer, executionState.collectedToolCalls, nil
}

func (g *graph) compile(executionState *graphExecutionState) (*langgraph.Runnable, error) {
	messageGraph := langgraph.NewMessageGraph()

	messageGraph.AddNode(graphNodeGuard, func(_ context.Context, messages []llms.MessageContent) ([]llms.MessageContent, error) {
		if hasInjections := g.guard.Check(executionState.userMessage); hasInjections {
			return messages, errPromptInjection
		}

		return messages, nil
	})
	messageGraph.AddNode(graphNodeRetrieve, func(ctx context.Context, messages []llms.MessageContent) ([]llms.MessageContent, error) {
		relevantServices, err := g.retriever.Retrieve(ctx, executionState.userMessage)
		if err != nil {
			return messages, fmt.Errorf("retrieve context: %w", err)
		}

		executionState.relevantServices = relevantServices
		return messages, nil
	})
	messageGraph.AddNode(graphNodePrepare, func(_ context.Context, state []llms.MessageContent) ([]llms.MessageContent, error) {
		messages := make([]modelMessage, 0, len(executionState.history)+4)
		messages = append(messages, modelMessage{
			Role:    "system",
			Content: buildSystemPrompt(g.config.SystemPrompt, g.allowedToolNames()),
		})

		if contextMessage := buildServicesContextMessage(executionState.relevantServices); contextMessage != "" {
			messages = append(messages, modelMessage{
				Role:    "system",
				Content: contextMessage,
			})
		}

		for _, turn := range executionState.history {
			messages = append(messages, modelMessage{
				Role:    turn.Role,
				Content: turn.Content,
			})
		}
		messages = append(messages, modelMessage{
			Role:    "user",
			Content: strings.TrimSpace(executionState.userMessage),
		})

		executionState.modelMessages = messages

		return state, nil
	})
	messageGraph.AddNode(
		graphNodeGenerateAnswer,
		func(ctx context.Context, messages []llms.MessageContent) ([]llms.MessageContent, error) {
			allowedToolDefinitions := g.allowedToolDefinitions()

			for i := 0; i < maxToolIterations; i++ {
				completion, completionErr := g.chat.complete(ctx, modelRequest{
					Model:       g.config.ModelName,
					Temperature: g.config.Temperature,
					MaxTokens:   g.config.MaxTokens,
					Messages:    executionState.modelMessages,
					Tools:       allowedToolDefinitions,
				})
				if completionErr != nil {
					return messages, fmt.Errorf("chat completion: %w", completionErr)
				}

				if len(completion.ToolCalls) == 0 {
					executionState.answer = strings.TrimSpace(completion.Content)
					return messages, nil
				}

				executionState.modelMessages = append(executionState.modelMessages, modelMessage{
					Role:      "assistant",
					Content:   completion.Content,
					ToolCalls: completion.ToolCalls,
				})

				for _, modelToolCall := range completion.ToolCalls {
					toolCallInfo := ToolCall{
						Name:      modelToolCall.Name,
						Arguments: modelToolCall.Arguments,
					}

					toolResultMessage := ""
					if !g.isToolAllowed(modelToolCall.Name) {
						toolCallInfo.Error = "tool is not allowed by assistant.tools configuration"
						toolResultMessage = toolCallInfo.Error
					} else {
						arguments, decodeErr := decodeToolArguments(modelToolCall.Arguments)
						if decodeErr != nil {
							toolCallInfo.Error = decodeErr.Error()
							toolResultMessage = decodeErr.Error()
						} else {
							result, runErr := g.tools.Execute(ctx, modelToolCall.Name, arguments)
							if runErr != nil {
								toolCallInfo.Error = runErr.Error()
								toolResultMessage = runErr.Error()
							} else {
								toolCallInfo.Result = result
								toolResultMessage = result
							}
						}
					}

					executionState.collectedToolCalls = append(executionState.collectedToolCalls, toolCallInfo)
					executionState.modelMessages = append(executionState.modelMessages, modelMessage{
						Role:       "tool",
						Name:       modelToolCall.Name,
						ToolCallID: modelToolCall.ID,
						Content:    strings.TrimSpace(toolResultMessage),
					})
				}
			}

			return messages, fmt.Errorf("tool iteration limit exceeded")
		},
	)

	messageGraph.AddEdge(graphNodeGuard, graphNodeRetrieve)
	messageGraph.AddEdge(graphNodeRetrieve, graphNodePrepare)
	messageGraph.AddEdge(graphNodePrepare, graphNodeGenerateAnswer)
	messageGraph.AddEdge(graphNodeGenerateAnswer, langgraph.END)
	messageGraph.SetEntryPoint(graphNodeGuard)

	return messageGraph.Compile()
}

func (g *graph) allowedToolDefinitions() []ToolDefinition {
	definitions := g.tools.Definitions()
	if len(g.allowedToolSet) == 0 {
		return definitions
	}

	filtered := make([]ToolDefinition, 0, len(definitions))
	for _, definition := range definitions {
		if g.isToolAllowed(definition.Name) {
			filtered = append(filtered, definition)
		}
	}

	return filtered
}

func (g *graph) allowedToolNames() []string {
	definitions := g.allowedToolDefinitions()
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.Name)
	}

	return names
}

func (g *graph) isToolAllowed(toolName string) bool {
	if len(g.allowedToolSet) == 0 {
		return true
	}

	_, ok := g.allowedToolSet[toolName]
	return ok
}

func decodeToolArguments(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("decode tool arguments: %w", err)
	}
	if decoded == nil {
		return map[string]any{}, nil
	}

	return decoded, nil
}

func buildServicesContextMessage(services []service.Info) string {
	if len(services) == 0 {
		return "No service metadata is available in service.store."
	}

	builder := strings.Builder{}
	builder.WriteString("Relevant service metadata from service.store:\n")
	for _, serviceInfo := range services {
		builder.WriteString("- ")
		builder.WriteString(serviceToContextDocument(serviceInfo))
		builder.WriteByte('\n')
	}

	return strings.TrimSpace(builder.String())
}

func serviceToContextDocument(serviceInfo service.Info) string {
	return strings.TrimSpace(
		fmt.Sprintf(
			"stack=%s service=%s type=%s image=%s description=%s",
			serviceInfo.Stack,
			serviceInfo.Name,
			serviceInfo.Type,
			serviceInfo.Image,
			serviceInfo.Description,
		),
	)
}
