# [Discuss] Core Feature Proposal: Topology-Aware "Blast Radius" Context Builder

Hi team! While reviewing `app/pipeline/pipeline.py`, I noticed how `node_correlate_upstream` handles correlation by extracting `upstream_services` from the raw alert and firing Datadog queries. I’ve been researching infrastructure graphs for my own AI-SRE project ([Claritty](https://github.com/Vaishnav88sk/claritty)), and I think OpenSRE can take this a massive step further.

### The Problem
Relying strictly on the raw alert's `upstream_services` leaves the `ConnectedInvestigationAgent` with limited spatial awareness. If an alert fires for "Checkout Service", the AI doesn't inherently know that Checkout relies on a specific AWS Subnet or Redis cluster unless it's explicitly stated in the alert payload. Without this structural context, the AI wastes tokens and time searching blindly.

### The Proposal: Architecture Context Builder
I propose we build an ingestion stage in `run_connected_investigation` that runs *before* the `ConnectedInvestigationAgent`. 

This stage would dynamically ingest Kubernetes manifests, Terraform state, or OpenTelemetry dependency data (via the `resolve_integrations` step) to build a lightweight, in-memory Knowledge Graph of the infrastructure topology inside `AgentState`.

### The Flow
1. An alert fires for `Payment API`.
2. The Topology Builder queries the graph and injects the precise blast radius into `AgentState`:
   `Payment API -> depends on -> PostgresDB_2 -> running on -> NodePool_B`. 
3. When `ConnectedInvestigationAgent` starts, its prompt and search space are mathematically constrained to *only* those components. 

### Why this matters for OpenSRE
Context is everything. By building the connective tissue that helps the AI understand *how* the system is wired, we drastically reduce LLM hallucinations, save tokens, and make the root-cause analysis 10x faster and more accurate.

I would love to draft the initial architecture for building this topology graph in `AgentState`. What are your thoughts on integrating deeper infrastructure context into the core roadmap?
