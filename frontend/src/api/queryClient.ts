import { QueryClient } from "@tanstack/react-query";

import { isApiError } from "./errors";

const MAX_RETRIES = 2;

function isClientError(error: unknown): boolean {
  return isApiError(error) && error.status >= 400 && error.status < 500;
}

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: (failureCount, error) => !isClientError(error) && failureCount < MAX_RETRIES,
        refetchOnWindowFocus: false,
      },
      mutations: {
        retry: false,
      },
    },
  });
}
