"use client";
import { NamespaceSelector } from "@/components/namespace-selector";
import { Button } from "@/components/ui/button";

import type { ReactElement } from "react";
import { TopologySelector } from "@/components/topology-selector.tsx";

interface VisualizeHeaderProps {
  readonly namespace: string;
  readonly topologyName: string;
  readonly setNamespace: (namespace: string) => void;
  readonly setTopologyName: (topoloygName: string) => void;
  readonly setTriggerDraw: (state: boolean) => void;
}

export function VisualizeHeader(props: VisualizeHeaderProps): ReactElement {
  const { namespace, topologyName, setNamespace, setTopologyName, setTriggerDraw } = props;

  return (
    <div className="flex w-full items-center justify-center">
      <div className="grid grid-cols-7 space-x-2 p-4">
        <NamespaceSelector
          namespace={namespace}
          placeholder="Select a namespace..."
          setNamespace={setNamespace}
        />
        <TopologySelector
          namespace={namespace}
          topologyName={topologyName}
          placeholder="Select a topology..."
          setTopologyName={setTopologyName}
        />
        <Button
          disabled={!namespace}
          onClick={(): void => {
            setTriggerDraw(true);
          }}
          type="submit"
        >
          Visualize
        </Button>
      </div>
    </div>
  );
}
