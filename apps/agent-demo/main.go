// agent-demo: A lightweight AI agent that performs a multi-step research task
// and records every step to the 0G Memory Hub orchestrator.
//
// It demonstrates the full OpenClaw event lifecycle:
//   agent_start → tool_call → tool_result → reasoning → agent_complete
//
// Usage:
//   ORCH_URL=http://127.0.0.1:8080 go run .
//   ORCH_URL=http://127.0.0.1:8080 OPENAI_API_KEY=sk-... go run .
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// ── Configuration ──

type config struct {
	OrchURL      string
	OpenAIKey    string
	OpenAIModel  string
	WorkflowID   string
	RunID        string
	AgentID      string
	UseMockLLM   bool
}

func loadConfig() config {
	orchURL := os.Getenv("ORCH_URL")
	if orchURL == "" {
		orchURL = "http://127.0.0.1:8080"
	}

	ts := time.Now().Format("20060102150405")
	runID := fmt.Sprintf("agent-demo-%s", ts)

	apiKey := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	return config{
		OrchURL:    orchURL,
		OpenAIKey:  apiKey,
		OpenAIModel: model,
		WorkflowID: "wf-" + runID,
		RunID:      runID,
		AgentID:    "agent-demo",
		UseMockLLM: apiKey == "",
	}
}

// ── Orchestrator client ──

type orchEvent struct {
	EventID    string `json:"eventId"`
	WorkflowID string `json:"workflowId"`
	RunID      string `json:"runId"`
	SessionID  string `json:"sessionId"`
	TraceID    string `json:"traceId"`
	EventType  string `json:"eventType"`
	Actor      string `json:"actor"`
	Role       string `json:"role"`
	ToolCallID string `json:"toolCallId,omitempty"`
	SkillName  string `json:"skillName,omitempty"`
	TaskID     string `json:"taskId,omitempty"`
	Payload    any    `json:"payload"`
}

func ingest(ctx context.Context, cfg config, evt orchEvent) error {
	body, _ := json.Marshal(evt)
	req, err := http.NewRequestWithContext(ctx, "POST", cfg.OrchURL+"/v1/openclaw/ingest", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("ingest %s: %w", evt.EventType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ingest %s: HTTP %d: %s", evt.EventType, resp.StatusCode, string(respBody))
	}
	return nil
}

func newEvent(cfg config, eventType, actor, role string, payload any) orchEvent {
	return orchEvent{
		EventID:    fmt.Sprintf("%s-%s-%d", cfg.RunID, eventType, time.Now().UnixMilli()),
		WorkflowID: cfg.WorkflowID,
		RunID:      cfg.RunID,
		SessionID:  "session-" + cfg.RunID,
		TraceID:    "trace-" + cfg.RunID,
		EventType:  eventType,
		Actor:      actor,
		Role:       role,
		Payload:    payload,
	}
}

// ── LLM interface ──

type llmResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	TokensPrompt int    `json:"tokens_prompt"`
	TokensCompl  int    `json:"tokens_completion"`
	LatencyMs    int64  `json:"latency_ms"`
}

func callLLM(ctx context.Context, cfg config, systemPrompt, userPrompt string) (llmResponse, error) {
	if cfg.UseMockLLM {
		return mockLLM(systemPrompt, userPrompt), nil
	}
	return callOpenAI(ctx, cfg, systemPrompt, userPrompt)
}

func mockLLM(system, user string) llmResponse {
	responses := map[string]string{
		"search": "Found 3 relevant results about 0G Network:\n1. 0G is a modular AI chain with decentralized storage\n2. 0G Storage provides erasure-coded blob storage on Galileo testnet\n3. 0G KV layer offers millisecond key-value queries over the storage layer",
		"analyze": "Key findings:\n- 0G uses a three-layer architecture: Storage (blobs), Chain (anchoring), KV (fast queries)\n- The Galileo testnet supports EVM-compatible smart contracts\n- Data availability is ensured through erasure coding and validator consensus",
		"summarize": "0G Network is a modular blockchain infrastructure designed for AI workloads. It provides decentralized storage with sub-second latency, on-chain data anchoring for integrity verification, and a KV layer for real-time queries. The three-layer stack enables durable, verifiable memory for autonomous agents.",
	}

	key := "summarize"
	if strings.Contains(strings.ToLower(user), "search") {
		key = "search"
	} else if strings.Contains(strings.ToLower(user), "analy") {
		key = "analyze"
	}

	latency := 200 + rand.Intn(800)
	time.Sleep(time.Duration(latency) * time.Millisecond)

	return llmResponse{
		Content:      responses[key],
		Model:        "mock-gpt-4o-mini",
		TokensPrompt: 50 + rand.Intn(100),
		TokensCompl:  80 + rand.Intn(120),
		LatencyMs:    int64(latency),
	}
}

func callOpenAI(ctx context.Context, cfg config, systemPrompt, userPrompt string) (llmResponse, error) {
	reqBody := map[string]any{
		"model": cfg.OpenAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens": 300,
	}
	body, _ := json.Marshal(reqBody)

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return llmResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return llmResponse{}, fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()
	latency := time.Since(start).Milliseconds()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return llmResponse{}, fmt.Errorf("OpenAI HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return llmResponse{}, err
	}

	content := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return llmResponse{
		Content:      content,
		Model:        result.Model,
		TokensPrompt: result.Usage.PromptTokens,
		TokensCompl:  result.Usage.CompletionTokens,
		LatencyMs:    latency,
	}, nil
}

// ── Agent task definition ──

type agentStep struct {
	Name       string
	SkillName  string
	TaskID     string
	System     string
	User       string
}

func researchSteps(topic string) []agentStep {
	return []agentStep{
		{
			Name:      "search",
			SkillName: "web_search",
			TaskID:    "task-search",
			System:    "You are a research assistant. Search for information and return structured results.",
			User:      fmt.Sprintf("Search for: %s", topic),
		},
		{
			Name:      "analyze",
			SkillName: "text_analysis",
			TaskID:    "task-analyze",
			System:    "You are an analyst. Analyze the search results and extract key findings.",
			User:      "Analyze the search results about 0G Network and identify the key technical components and their relationships.",
		},
		{
			Name:      "summarize",
			SkillName: "text_generation",
			TaskID:    "task-summarize",
			System:    "You are a technical writer. Create a concise summary from the analysis.",
			User:      "Write a 2-3 sentence summary of 0G Network's architecture and value proposition for AI agents.",
		},
	}
}

// ── Main ──

func main() {
	cfg := loadConfig()
	ctx := context.Background()

	fmt.Printf("=== 0G Memory Hub Agent Demo ===\n")
	fmt.Printf("orchestrator: %s\n", cfg.OrchURL)
	fmt.Printf("workflow:     %s\n", cfg.WorkflowID)
	fmt.Printf("run:          %s\n", cfg.RunID)
	if cfg.UseMockLLM {
		fmt.Printf("llm:          mock (set OPENAI_API_KEY for real LLM)\n")
	} else {
		fmt.Printf("llm:          %s (OpenAI)\n", cfg.OpenAIModel)
	}
	fmt.Println()

	topic := "0G Network decentralized AI infrastructure"
	steps := researchSteps(topic)

	// 1. agent_start
	fmt.Printf("[1/%d] agent_start\n", len(steps)+2)
	err := ingest(ctx, cfg, newEvent(cfg, "agent_start", cfg.AgentID, "orchestrator", map[string]any{
		"task":        "research",
		"topic":       topic,
		"step_count":  len(steps),
		"llm_backend": ternary(cfg.UseMockLLM, "mock", cfg.OpenAIModel),
	}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// 2. Execute each step: tool_call → LLM → tool_result → reasoning
	for i, step := range steps {
		stepNum := i + 2
		toolCallID := fmt.Sprintf("%s-call-%d", cfg.RunID, i)

		// tool_call
		fmt.Printf("[%d/%d] tool_call: %s\n", stepNum, len(steps)+2, step.SkillName)
		evt := newEvent(cfg, "tool_call", cfg.AgentID, "planner", map[string]any{
			"skill":  step.SkillName,
			"prompt": step.User,
		})
		evt.ToolCallID = toolCallID
		evt.SkillName = step.SkillName
		evt.TaskID = step.TaskID
		if err := ingest(ctx, cfg, evt); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}

		// Call LLM
		llmResp, err := callLLM(ctx, cfg, step.System, step.User)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR calling LLM: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("         llm: %s (%d+%d tokens, %dms)\n",
			llmResp.Model, llmResp.TokensPrompt, llmResp.TokensCompl, llmResp.LatencyMs)

		// tool_result
		evt = newEvent(cfg, "tool_result", cfg.AgentID, "executor", map[string]any{
			"skill":             step.SkillName,
			"result":            llmResp.Content,
			"model":             llmResp.Model,
			"tokens_prompt":     llmResp.TokensPrompt,
			"tokens_completion": llmResp.TokensCompl,
			"latency_ms":        llmResp.LatencyMs,
		})
		evt.ToolCallID = toolCallID
		evt.SkillName = step.SkillName
		evt.TaskID = step.TaskID
		if err := ingest(ctx, cfg, evt); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}

		// reasoning (agent's internal thought after receiving result)
		evt = newEvent(cfg, "reasoning", cfg.AgentID, "planner", map[string]any{
			"thought": fmt.Sprintf("Step '%s' completed successfully. %s",
				step.Name, ternary(i < len(steps)-1, "Proceeding to next step.", "All steps done.")),
		})
		evt.TaskID = step.TaskID
		if err := ingest(ctx, cfg, evt); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
	}

	// 3. agent_complete
	fmt.Printf("[%d/%d] agent_complete\n", len(steps)+2, len(steps)+2)
	err = ingest(ctx, cfg, newEvent(cfg, "agent_complete", cfg.AgentID, "orchestrator", map[string]any{
		"status":       "success",
		"steps_total":  len(steps),
		"events_total": len(steps)*3 + 2,
	}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("=== Done ===\n")
	fmt.Printf("Events ingested: %d\n", len(steps)*3+2)
	fmt.Printf("Verify:    curl %s/v1/openclaw/runs/%s/verify | jq\n", cfg.OrchURL, cfg.RunID)
	fmt.Printf("Trace:     curl %s/v1/openclaw/runs/%s/trace  | jq\n", cfg.OrchURL, cfg.RunID)
	fmt.Printf("Dashboard: %s/dashboard\n", cfg.OrchURL)
	fmt.Printf("Judge:     %s/judge/verify?runId=%s\n", cfg.OrchURL, cfg.RunID)
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
