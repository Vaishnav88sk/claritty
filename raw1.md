# [Discuss] Core Feature Proposal: Executable AI Runbooks (Bridging the Trust Gap)

Hi team! While contributing to the integrations, I spent some time analyzing `app/pipeline/pipeline.py` and the `ConnectedInvestigationAgent`. I’ve been exploring AI-SRE adoption hurdles for my own open-source project ([Claritty](https://github.com/Vaishnav88sk/claritty)), and I believe OpenSRE has the perfect pipeline architecture to solve the biggest one: **Trust and Determinism**.

### The Problem
Currently, the `ConnectedInvestigationAgent` relies heavily on free-form LLM reasoning to decide which tools to call during an incident. This is great for exploration, but in enterprise environments, companies want the AI to strictly follow their existing, approved Standard Operating Procedures (SOPs) rather than "winging it." 

### The Proposal: Runbook-as-Code
I propose we introduce a **Markdown Runbook Execution Engine** as a core mode within or alongside `ConnectedInvestigationAgent`. 

**How it integrates:**
1. During `extract_alert(state)`, if the alert matches a known routing key, OpenSRE fetches the corresponding `.md` runbook from a configured GitHub repository.
2. We build a parser that converts the Markdown document into a deterministic execution graph.
3. Instead of a standard ReAct loop, the agent treats the Markdown file as a strict state machine—parsing out bash commands, Datadog queries, or metrics links embedded in the text.
4. It executes the SOP step-by-step, halting via `app/guardrails` or `app/sandbox` for human approval before executing any destructive remediation actions.

### Why this matters for OpenSRE
This solves the "Zero-to-Value" problem. Enterprises won't adopt AI SRE if they have to train the AI from scratch. If they can point OpenSRE at their existing repo of markdown runbooks, adoption becomes instant and inherently trusted.

I would love to take the lead on architecting the Markdown parser and state execution. What are your thoughts on integrating this determinism into the core pipeline?
