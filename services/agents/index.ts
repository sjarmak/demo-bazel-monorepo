/**
 * Agent Integration Layer
 *
 * Provides infrastructure for AI coding agents to interact with the platform.
 * Supports context retrieval, workflow execution, and security validation.
 */

export { AgentOrchestrator } from './orchestrator';
export { AgentRegistry, RegisterAgent, type AgentConfig } from './registry';
export { AgentContext, createAgentContext } from './context';
export { authMiddleware, rateLimitMiddleware, loggingMiddleware } from './middleware';
