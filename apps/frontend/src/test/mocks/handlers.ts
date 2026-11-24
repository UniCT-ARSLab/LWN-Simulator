import type { RequestHandler } from "msw";

/**
 * Default HTTP handlers used by frontend tests.
 *
 * Feature-specific handlers should be added in the test that needs them with
 * `server.use(...)`, keeping the global test environment deliberately empty.
 */
export const handlers: RequestHandler[] = [];
