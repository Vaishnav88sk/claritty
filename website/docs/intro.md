---
sidebar_position: 1
---

# Introduction

**Claritty** is a production-grade, open-source AI Site Reliability Engineering (SRE) platform specifically designed for Kubernetes.

It combines real-time cluster telemetry with a powerful **6-stage AI agent pipeline** to automatically detect, diagnose, and remediate incidents, shrinking MTTR (Mean Time to Resolution) from hours down to minutes.

:::tip

Claritty is designed with **Zero-Trust** in mind. By leveraging local LLMs (like Ollama), your sensitive cluster telemetry and logs never leave your infrastructure.

:::

## The Two Modes of Claritty

Claritty provides two powerful ways to interact with your Kubernetes infrastructure, depending on your needs:

### 1. Clarctl CLI (Local Tool)
A powerful command-line interface run from your local machine. It connects to your current Kubernetes context to instantly analyze namespaces or specific pods, generate an RCA (Root Cause Analysis), and offer interactive, step-by-step remediation commands. **Perfect for on-call engineers debugging live incidents.**

### 2. SRE Agent & Hub (In-Cluster Platform)
A lightweight, in-cluster daemon (the Agent) that continuously monitors your infrastructure. It autonomously performs the 6-stage AI pipeline on failing resources and pushes structured incident reports to a centralized Hub server. The Hub provides a beautiful web dashboard for a multi-cluster overview, Slack alerts, and detailed RCA records. **Perfect for continuous production monitoring.**

## Key Features

- 📊 **Node & Pod-Level Telemetry**: Real-time resource usage and metrics collection.
- ⚡ **Auto Incident Detection**: Detects cascading failures, API throttling, Split-Brain StatefulSets, network partitions, and more.
- 🧠 **6-Stage AI Pipeline**: Triage, Metrics, Logs, Infra, Runbook, and Commander agents collaboratively diagnose root causes.
- 🚨 **Interactive Auto-Remediation**: Step-by-step CLI prompts (`y / dry / n`) before executing any fix.
- 🌐 **Centralized Dashboard**: A web UI to view multi-cluster health, active incidents, and automated remediation plans.
- 🔒 **Safety First**: Destructive commands are flagged. All fixes are verified against a strict allowlist.
