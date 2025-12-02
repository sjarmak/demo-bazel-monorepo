/**
 * Agent Context
 *
 * Provides context management for agent sessions.
 * Tracks session state, permissions, and activity.
 */

export interface AgentContext {
  agentId: string;
  sessionId: string;
  permissions: string[];
  createdAt: Date;
  expiresAt: Date;
  metadata?: Record<string, unknown>;
}

export interface CreateContextOptions {
  agentId: string;
  permissions?: string[];
  ttlMinutes?: number;
  metadata?: Record<string, unknown>;
}

/**
 * Create a new agent context for a session.
 *
 * @param options - Options for context creation
 * @returns A new AgentContext
 *
 * @example
 * const ctx = createAgentContext({
 *   agentId: 'agent-claude-abc123',
 *   permissions: ['read', 'write', 'security:scan:execute'],
 *   ttlMinutes: 60,
 * });
 */
export function createAgentContext(options: CreateContextOptions): AgentContext {
  const now = new Date();
  const ttl = options.ttlMinutes ?? 30;

  return {
    agentId: options.agentId,
    sessionId: generateSessionId(),
    permissions: options.permissions ?? ['read'],
    createdAt: now,
    expiresAt: new Date(now.getTime() + ttl * 60 * 1000),
    metadata: options.metadata,
  };
}

/**
 * Check if a context is still valid (not expired).
 */
export function isContextValid(context: AgentContext): boolean {
  return new Date() < context.expiresAt;
}

/**
 * Extend the expiration of a context.
 */
export function extendContext(context: AgentContext, additionalMinutes: number): AgentContext {
  return {
    ...context,
    expiresAt: new Date(context.expiresAt.getTime() + additionalMinutes * 60 * 1000),
  };
}

/**
 * Check if context has a specific permission.
 */
export function contextHasPermission(context: AgentContext, permission: string): boolean {
  // Check for wildcard permissions
  if (permission.includes(':')) {
    const [category] = permission.split(':');
    if (context.permissions.includes(`${category}:*`)) {
      return true;
    }
  }

  return context.permissions.includes(permission);
}

function generateSessionId(): string {
  const timestamp = Date.now().toString(36);
  const random = Math.random().toString(36).substring(2, 10);
  return `session-${timestamp}-${random}`;
}
