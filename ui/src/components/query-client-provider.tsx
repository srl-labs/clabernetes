"use client";

import { getQueryClient } from "@/lib/get-query-client";
import { QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import type { ReactElement, ReactNode } from "react";

interface QueryClientProviderWrapperProps {
  readonly children: ReactNode;
}

export function QueryClientProviderWrapper({
  children,
}: QueryClientProviderWrapperProps): ReactElement {
  const queryClient = getQueryClient();

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
