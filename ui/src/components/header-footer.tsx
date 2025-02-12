"use client";
import { ThemeToggle } from "@/components/theme-toggle";
import type { ReactElement } from "react";
import { Button } from "@/components/ui/button.tsx";
import Link from "next/link";
import Image from "next/image";
import C9sLogo from "@/public/c9s-logo.png";
import ContainerlabLogo from "@/public/clab-logo.svg";
import { usePathname } from "next/navigation";

const navigationMenu =
  "group inline-flex h-10 w-48 items-center justify-center rounded-md bg-background px-4 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground focus:outline-hidden disabled:pointer-events-none disabled:opacity-50 data-active:bg-accent/50 data-[state=open]:bg-accent/50";
const navigationMenuTriggered =
  "group inline-flex h-10 w-48 items-center justify-center rounded-md bg-background px-4 py-2 text-sm font-medium transition-colors bg-accent bg-accent text-accent-foreground outline-hidden opacity-50 bg-accent/50 data-[state=open]:bg-accent/5";

function getNavButtonCss(buttonIsActivePath: boolean): string {
  if (buttonIsActivePath) {
    return navigationMenuTriggered;
  }
  return navigationMenu;
}

interface HeaderFooterProps {
  readonly children: ReactElement;
}

export function HeaderFooter(props: HeaderFooterProps): ReactElement {
  const { children } = props;

  const currentPath = usePathname();

  return (
    <div className="flex h-screen flex-col">
      <div className="bg-primary-foreground w-full flex items-center justify-center p-8 shadow-md">
        <Image
          src={C9sLogo}
          height={64}
          width={64}
          alt="clabernetes super cool logo"
        />
        <h1 className="font-heading text-3xl md:text-4xl">clabernetes</h1>
      </div>
      <div className="flex items-center justify-center space-x-4 px-4 pt-4">
        <Button
          asChild={true}
          className={getNavButtonCss(currentPath === "/topologies")}
          disabled={currentPath === "/topologies"}
          variant="outline"
        >
          <Link href="/topologies">Topologies</Link>
        </Button>
        <Button
          asChild={true}
          className={getNavButtonCss(currentPath === "/visualizer")}
          disabled={currentPath === "/visualizer"}
          variant="outline"
        >
          <Link href="/visualizer">Visualizer</Link>
        </Button>
      </div>
      <div className="container mx-auto flex justify-center">{children}</div>
      <div className="bg-primary-foreground fixed bottom-0 w-full">
        <div className="fixed bottom-0 mt-auto p-2">
          <ThemeToggle />
        </div>
        <div className="container mx-auto flex items-center justify-center p-2">
          <Image
            className="stroke-white hidden lg:block"
            src={ContainerlabLogo}
            height={64}
            width={64}
            alt="containerlabs super cool logo"
          />
        </div>
      </div>
    </div>
  );
}
