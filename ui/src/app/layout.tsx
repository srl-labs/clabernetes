import type { Metadata } from "next";
import "./globals.css";
import { Inter as fontSans } from "next/font/google";
import type React from "react";
import type { ReactElement } from "react";
import { ThemeProvider as NextThemesProvider } from "next-themes";
import { cn } from "@/lib/utils";
import { HeaderFooter } from "@/components/header-footer.tsx";
import { QueryClientProviderWrapper } from "@/components/query-client-provider.tsx";

const font = fontSans({
  subsets: ["latin"],
  variable: "--font-sans",
});

export const metadata: Metadata = {
  title: "clabernetes",
  description: "containerlab, just in kubernetes!",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>): ReactElement {
  return (
    <html
      lang="en"
      suppressHydrationWarning={true}
    >
      <head />
      <body className={cn("bg-background font-sans antialiased", font.variable)}>
        <NextThemesProvider
          attribute="class"
          defaultTheme="system"
          disableTransitionOnChange={true}
          enableSystem={true}
        >
          <HeaderFooter>
            <QueryClientProviderWrapper>{children}</QueryClientProviderWrapper>
          </HeaderFooter>
        </NextThemesProvider>
      </body>
    </html>
  );
}
