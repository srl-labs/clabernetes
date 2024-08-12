import { defaultShouldDehydrateQuery, isServer, QueryClient } from "@tanstack/react-query";

const secondsInMilliSeconds = 1000;
const secondsInMinute = 60;
const staleTime = secondsInMinute * secondsInMilliSeconds;

function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      dehydrate: {
        shouldDehydrateQuery: (query): boolean => {
          return defaultShouldDehydrateQuery(query) || query.state.status === "pending";
        },
      },
      queries: {
        staleTime: staleTime,
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined;

export function getQueryClient(): QueryClient {
  if (isServer) {
    return makeQueryClient();
  }

  if (!browserQueryClient) {
    browserQueryClient = makeQueryClient();
  }

  return browserQueryClient;
}
