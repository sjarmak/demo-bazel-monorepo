/**
 * Agent Middleware
 *
 * Express-style middleware for handling agent requests.
 * Provides authentication, rate limiting, and logging.
 */

import { AgentContext, isContextValid } from './context';
import { AgentRegistry } from './registry';

export interface AgentRequest {
  context: AgentContext;
  operation: string;
  payload: unknown;
  timestamp: Date;
}

export interface AgentResponse {
  success: boolean;
  data?: unknown;
  error?: string;
  requestId: string;
}

export type NextFunction = () => Promise<AgentResponse>;
export type Middleware = (req: AgentRequest, next: NextFunction) => Promise<AgentResponse>;

/**
 * Authentication middleware.
 * Validates that the agent is registered and the session is valid.
 */
export const authMiddleware: Middleware = async (req, next) => {
  const { context } = req;

  // Check if agent is registered
  const agent = AgentRegistry.getAgent(context.agentId);
  if (!agent) {
    return {
      success: false,
      error: `Agent not found: ${context.agentId}`,
      requestId: generateRequestId(),
    };
  }

  // Check if session is valid
  if (!isContextValid(context)) {
    return {
      success: false,
      error: 'Session expired. Please create a new context.',
      requestId: generateRequestId(),
    };
  }

  return next();
};

/**
 * Rate limiting middleware.
 * Enforces request limits per agent.
 */
const requestCounts = new Map<string, { count: number; windowStart: number }>();

export const rateLimitMiddleware: Middleware = async (req, next) => {
  const { context } = req;
  const agent = AgentRegistry.getAgent(context.agentId);

  if (!agent) {
    return next();
  }

  const rateLimit = agent.config.rateLimit ?? { requestsPerMinute: 60, burstLimit: 10 };
  const now = Date.now();
  const windowMs = 60 * 1000;

  let record = requestCounts.get(context.agentId);

  // Reset window if expired
  if (!record || now - record.windowStart > windowMs) {
    record = { count: 0, windowStart: now };
  }

  record.count++;
  requestCounts.set(context.agentId, record);

  if (record.count > rateLimit.requestsPerMinute) {
    return {
      success: false,
      error: `Rate limit exceeded. Max ${rateLimit.requestsPerMinute} requests per minute.`,
      requestId: generateRequestId(),
    };
  }

  return next();
};

/**
 * Logging middleware.
 * Logs all agent requests for audit trail.
 */
export const loggingMiddleware: Middleware = async (req, next) => {
  const requestId = generateRequestId();
  const startTime = Date.now();

  console.log(
    JSON.stringify({
      event: 'agent_request_start',
      requestId,
      agentId: req.context.agentId,
      sessionId: req.context.sessionId,
      operation: req.operation,
      timestamp: req.timestamp.toISOString(),
    })
  );

  try {
    const response = await next();

    console.log(
      JSON.stringify({
        event: 'agent_request_complete',
        requestId,
        agentId: req.context.agentId,
        operation: req.operation,
        success: response.success,
        durationMs: Date.now() - startTime,
      })
    );

    return { ...response, requestId };
  } catch (error) {
    console.error(
      JSON.stringify({
        event: 'agent_request_error',
        requestId,
        agentId: req.context.agentId,
        operation: req.operation,
        error: error instanceof Error ? error.message : 'Unknown error',
        durationMs: Date.now() - startTime,
      })
    );

    throw error;
  }
};

/**
 * Permission checking middleware factory.
 *
 * @param requiredPermission - The permission required for this operation
 */
export function requirePermission(requiredPermission: string): Middleware {
  return async (req, next) => {
    const { context } = req;

    // Check for exact match
    if (context.permissions.includes(requiredPermission)) {
      return next();
    }

    // Check for wildcard match
    if (requiredPermission.includes(':')) {
      const [category] = requiredPermission.split(':');
      if (context.permissions.includes(`${category}:*`)) {
        return next();
      }
    }

    return {
      success: false,
      error: `Permission denied: ${requiredPermission}`,
      requestId: generateRequestId(),
    };
  };
}

/**
 * Compose multiple middleware into a single function.
 */
export function composeMiddleware(...middlewares: Middleware[]): Middleware {
  return async (req, finalNext) => {
    let index = -1;

    const dispatch = async (i: number): Promise<AgentResponse> => {
      if (i <= index) {
        throw new Error('next() called multiple times');
      }
      index = i;

      if (i === middlewares.length) {
        return finalNext();
      }

      const middleware = middlewares[i];
      return middleware(req, () => dispatch(i + 1));
    };

    return dispatch(0);
  };
}

function generateRequestId(): string {
  return `req-${Date.now().toString(36)}-${Math.random().toString(36).substring(2, 8)}`;
}
