/**
 * Agent Registry
 *
 * Manages registration and lifecycle of AI coding agents.
 * Each agent must register before making requests to the platform.
 */

export interface AgentConfig {
  name: string;
  version: string;
  permissions: Permission[];
  rateLimit?: RateLimitConfig;
  metadata?: Record<string, string>;
}

export type Permission =
  | 'read'
  | 'write'
  | 'execute'
  | 'security'
  | 'security:scan:execute'
  | 'security:*'
  | 'workflow:trigger'
  | 'context:retrieve';

export interface RateLimitConfig {
  requestsPerMinute: number;
  burstLimit: number;
}

export interface RegisteredAgent {
  id: string;
  config: AgentConfig;
  registeredAt: Date;
  lastActiveAt: Date;
  sessionCount: number;
}

class AgentRegistryImpl {
  private agents: Map<string, RegisteredAgent> = new Map();

  /**
   * Register a new agent with the platform.
   *
   * @param config - Agent configuration including name, version, and permissions
   * @returns The registered agent with generated ID
   *
   * @example
   * const agent = registry.register({
   *   name: 'claude-code',
   *   version: '1.0.0',
   *   permissions: ['read', 'write', 'security:scan:execute'],
   * });
   */
  register(config: AgentConfig): RegisteredAgent {
    const id = this.generateAgentId(config.name);

    // Validate permissions
    this.validatePermissions(config.permissions);

    const agent: RegisteredAgent = {
      id,
      config,
      registeredAt: new Date(),
      lastActiveAt: new Date(),
      sessionCount: 0,
    };

    this.agents.set(id, agent);
    console.log(`Agent registered: ${id} with permissions: ${config.permissions.join(', ')}`);

    return agent;
  }

  /**
   * @deprecated Use register() instead. Will be removed in v3.0.
   */
  legacyRegisterAgent(name: string, permissions: string[]): RegisteredAgent {
    console.warn('legacyRegisterAgent is deprecated. Use register() instead.');
    return this.register({
      name,
      version: '0.0.0',
      permissions: permissions as Permission[],
    });
  }

  /**
   * Get an agent by ID.
   */
  getAgent(id: string): RegisteredAgent | undefined {
    const agent = this.agents.get(id);
    if (agent) {
      agent.lastActiveAt = new Date();
    }
    return agent;
  }

  /**
   * Check if an agent has a specific permission.
   */
  hasPermission(agentId: string, permission: Permission): boolean {
    const agent = this.agents.get(agentId);
    if (!agent) return false;

    const perms = agent.config.permissions;

    // Check for wildcard permissions
    if (perms.includes('security:*') && permission.startsWith('security:')) {
      return true;
    }

    return perms.includes(permission);
  }

  /**
   * Revoke an agent's registration.
   */
  revoke(agentId: string): boolean {
    return this.agents.delete(agentId);
  }

  /**
   * List all registered agents.
   */
  listAgents(): RegisteredAgent[] {
    return Array.from(this.agents.values());
  }

  private generateAgentId(name: string): string {
    const timestamp = Date.now().toString(36);
    const random = Math.random().toString(36).substring(2, 8);
    return `agent-${name}-${timestamp}-${random}`;
  }

  private validatePermissions(permissions: Permission[]): void {
    const validPermissions: Permission[] = [
      'read',
      'write',
      'execute',
      'security',
      'security:scan:execute',
      'security:*',
      'workflow:trigger',
      'context:retrieve',
    ];

    for (const perm of permissions) {
      if (!validPermissions.includes(perm)) {
        throw new Error(`Invalid permission: ${perm}`);
      }
    }
  }
}

// Singleton instance
export const AgentRegistry = new AgentRegistryImpl();

// Convenience function
export function RegisterAgent(config: AgentConfig): RegisteredAgent {
  return AgentRegistry.register(config);
}
