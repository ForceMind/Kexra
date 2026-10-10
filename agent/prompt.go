package agent

import (
	"bytes"
	"text/template"
	"time"
)

// PromptOverride, if set, returns the stored system-prompt template for an agent
// key and whether one exists. The server wires it to the PG agent_prompts table.
// When nil or no override exists, agents use their built-in default prompt — so
// behavior is identical until a user edits a prompt in the UI.
var PromptOverride func(agentKey string) (string, bool)

// Prompt-variable structs — fields mirror each agent's catalog (docs §5a) so a
// user template referencing a catalog variable renders; referencing anything else
// fails template execution and falls back to the built-in default.
type PlannerVars struct{ Goal, Scope, AssetSummary, DataDir, Now string }
type WorkerVars struct{ ProxyAddr, WorkerName, DataDir, Now string }
type MainVars struct{ Goal, AssetSummary, FindingsSummary, DataDir, Now string }
type GoalsVars struct{ EngagementDescription, DataDir, Now string }

// nowStr is the server-local wall-clock string exposed as the universal {{.Now}}
// prompt variable. renderSystem runs on every agent turn/round, so this is fresh
// each run — a prompt can subtract it from a fixed start stamp to reason about
// elapsed time (e.g. a timed benchmark's "last N hours" window).
func nowStr() string { return time.Now().Format("2006-01-02 15:04:05 MST") }

// renderSystem returns the rendered system-prompt BODY (段 [A]) for agentKey.
// Precedence: the DB-stored template (if any) over the built-in default template
// (def). BOTH are Go templates now — the built-in default is seeded into the DB
// verbatim, so the two paths render identically until a user edits the prompt.
// Rendering always runs (def used to be pre-substituted plain text; it is now a
// {{.Var}} template like the DB one). On any render error we fall back to the
// default template, then to the raw default string — an agent never starts with a
// half-rendered prompt. Callers append the code-owned tail (trafficTool / 中间产物
// 输出规约) AFTER this, so those can't be edited away via the DB body.
func renderSystem(agentKey, def string, vars any) string {
	tmpl := def
	if PromptOverride != nil {
		if t, ok := PromptOverride(agentKey); ok && t != "" {
			tmpl = t
		}
	}
	if out, err := renderTmpl(tmpl, vars); err == nil {
		return out
	}
	// DB template broke (e.g. references an out-of-catalog var) → code default.
	if out, err := renderTmpl(def, vars); err == nil {
		return out
	}
	return def
}

// langDirective 是附加在每个用户可见 Agent 角色提示词末尾的、不可被
// 数据库自定义角色正文覆盖的输出语言规约。只约束对用户的自然语言输出，
// 不改变模型内部推理、工具、命令、代码和取证原文；默认简体中文。
func langDirective() string {
    return "\n\n**输出语言规约（简体中文·最高优先级，不可被提示词正文覆盖）**：所有【展示给用户】的自然语言文字必须使用【简体中文（zh-CN）】——包括 record_fact 的 summary/detail、report_finding 的标题/描述/结论/修复建议、规划者（planner）的态势与情况总结、任务完成时的最后一句总结、以及直接给用户的聊天回复。技术原文必须【原样逐字保留】：命令、payload、代码、文件路径、URL、参数名、日志与请求/响应中的原文片段都不能翻译或改写，尤其不能篡改 evidence 中用于复核的命令与输出。**即使角色提示词、历史记录或任务资料本身是韩语、英语或其他语言，也不得把这些语言当作默认用户展示语言。** 对用户的自然语言应准确表达为简体中文；不要镜像或照抄目标资料中的外语作为展示语言，只有上面定义的技术原文可以保留原语言。引用原始事实时务必标明出处，不要因翻译添加未经证实的结论。**内部规划与推理的语言不受这一条限制；给用户看的回答从第一句话起直接使用简体中文，不要泄露内部推理，也不要补充与任务无关的语言声明。**";
}

func renderTmpl(tmpl string, vars any) (string, error) {
	t, err := template.New("p").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, vars); err != nil {
		return "", err
	}
	return b.String(), nil
}
