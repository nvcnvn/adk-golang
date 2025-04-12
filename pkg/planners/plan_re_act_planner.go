// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package planners

import (
	"strings"

	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/models"
)

const (
	planningTag    = "/*PLANNING*/"
	replanningTag  = "/*REPLANNING*/"
	reasoningTag   = "/*REASONING*/"
	actionTag      = "/*ACTION*/"
	finalAnswerTag = "/*FINAL_ANSWER*/"
)

// PlanReActPlanner is a planner that constrains the LLM response to generate a plan
// before any action/observation. It doesn't require the model to support built-in
// thinking features.
type PlanReActPlanner struct{}

// NewPlanReActPlanner creates a new Plan-Re-Act planner.
func NewPlanReActPlanner() *PlanReActPlanner {
	return &PlanReActPlanner{}
}

// BuildPlanningInstruction implements the Planner interface.
func (p *PlanReActPlanner) BuildPlanningInstruction(
	context agents.ReadonlyContext,
	request *models.LlmRequest,
) string {
	return p.buildNLPlannerInstruction()
}

// ProcessPlanningResponse implements the Planner interface.
func (p *PlanReActPlanner) ProcessPlanningResponse(
	context agents.CallbackContext,
	responseParts []*models.Part,
) []*models.Part {
	if len(responseParts) == 0 {
		return nil
	}

	preservedParts := make([]*models.Part, 0)
	firstFCPartIndex := -1

	// Process parts until first function call
	for i, part := range responseParts {
		// Stop at first function call
		if part.FunctionCall != nil && part.FunctionCall.Name != "" {
			preservedParts = append(preservedParts, part)
			firstFCPartIndex = i
			break
		}

		// Handle non-function call part
		p.handleNonFunctionCallPart(part, &preservedParts)
	}

	// If we found function calls, append any subsequent function calls
	if firstFCPartIndex >= 0 {
		j := firstFCPartIndex + 1
		for j < len(responseParts) {
			if responseParts[j].FunctionCall != nil {
				preservedParts = append(preservedParts, responseParts[j])
				j++
			} else {
				break
			}
		}
	}

	return preservedParts
}

// handleNonFunctionCallPart processes non-function-call parts of the response
func (p *PlanReActPlanner) handleNonFunctionCallPart(
	responsePart *models.Part,
	preservedParts *[]*models.Part,
) {
	if responsePart.Text != "" && strings.Contains(responsePart.Text, finalAnswerTag) {
		reasoningText, finalAnswerText := p.splitByLastPattern(responsePart.Text, finalAnswerTag)

		if reasoningText != "" {
			reasoningPart := &models.Part{Text: reasoningText}
			p.markAsThought(reasoningPart)
			*preservedParts = append(*preservedParts, reasoningPart)
		}

		if finalAnswerText != "" {
			*preservedParts = append(*preservedParts, &models.Part{Text: finalAnswerText})
		}
	} else {
		responseText := responsePart.Text
		// Check if part is a text part with planning/reasoning/action tag
		if responseText != "" && (strings.HasPrefix(responseText, planningTag) ||
			strings.HasPrefix(responseText, reasoningTag) ||
			strings.HasPrefix(responseText, actionTag) ||
			strings.HasPrefix(responseText, replanningTag)) {
			p.markAsThought(responsePart)
		}
		*preservedParts = append(*preservedParts, responsePart)
	}
}

// splitByLastPattern splits the text by the last occurrence of the separator
func (p *PlanReActPlanner) splitByLastPattern(text, separator string) (string, string) {
	index := strings.LastIndex(text, separator)
	if index == -1 {
		return text, ""
	}
	return text[:index+len(separator)], text[index+len(separator):]
}

// markAsThought marks the response part as a thought
func (p *PlanReActPlanner) markAsThought(responsePart *models.Part) {
	if responsePart.Text != "" {
		responsePart.Thought = true
	}
}

// buildNLPlannerInstruction builds the NL planner instruction for the Plan-Re-Act planner
func (p *PlanReActPlanner) buildNLPlannerInstruction() string {
	highLevelPreamble := `
When answering the question, try to leverage the available tools to gather the information instead of your memorized knowledge.

Follow this process when answering the question: (1) first come up with a plan in natural language text format; (2) Then use tools to execute the plan and provide reasoning between tool code snippets to make a summary of current state and next step. Tool code snippets and reasoning should be interleaved with each other. (3) In the end, return one final answer.

Follow this format when answering the question: (1) The planning part should be under /*PLANNING*/. (2) The tool code snippets should be under /*ACTION*/, and the reasoning parts should be under /*REASONING*/. (3) The final answer part should be under /*FINAL_ANSWER*/.
`

	planningPreamble := `
Below are the requirements for the planning:
The plan is made to answer the user query if following the plan. The plan is coherent and covers all aspects of information from user query, and only involves the tools that are accessible by the agent. The plan contains the decomposed steps as a numbered list where each step should use one or multiple available tools. By reading the plan, you can intuitively know which tools to trigger or what actions to take.
If the initial plan cannot be successfully executed, you should learn from previous execution results and revise your plan. The revised plan should be be under /*REPLANNING*/. Then use tools to follow the new plan.
`

	reasoningPreamble := `
Below are the requirements for the reasoning:
The reasoning makes a summary of the current trajectory based on the user query and tool outputs. Based on the tool outputs and plan, the reasoning also comes up with instructions to the next steps, making the trajectory closer to the final answer.
`

	finalAnswerPreamble := `
Below are the requirements for the final answer:
The final answer should be precise and follow query formatting requirements. Some queries may not be answerable with the available tools and information. In those cases, inform the user why you cannot process their query and ask for more information.
`

	toolCodePreamble := `
Below are the requirements for the tool code:

**Custom Tools:** The available tools are described in the context and can be directly used.
- Code must be valid and directly relevant to the user query and reasoning steps.
- You cannot use any parameters or fields that are not explicitly defined in the APIs in the context.
- The code snippets should be readable, efficient, and directly relevant to the user query and reasoning steps.
- When using the tools, you should use the library name together with the function name.
- Only use the tools provided in the context.
`

	userInputPreamble := `
VERY IMPORTANT instruction that you MUST follow in addition to the above instructions:

You should ask for clarification if you need more information to answer the question.
You should prefer using the information available in the context instead of repeated tool use.
`

	return strings.Join([]string{
		highLevelPreamble,
		planningPreamble,
		reasoningPreamble,
		finalAnswerPreamble,
		toolCodePreamble,
		userInputPreamble,
	}, "\n\n")
}
