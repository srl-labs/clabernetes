"use client";
import { type ReactElement, useState } from "react";
import useResizeObserver from "use-resize-observer";
import { ReactFlowProvider } from "@xyflow/react";
import { VisualizeFlow } from "@/components/visualizer/flow.tsx";
import { VisualizeHeader } from "@/components/visualizer/header.tsx";

interface Dimensions {
  height: number;
  width: number;
}

export function Visualizer(): ReactElement {
  const [flowDivSize, setFlowDivSize] = useState<Dimensions>({ height: 0, width: 0 });

  const [namespace, setNamespace] = useState<string>("");
  const [topologyName, setTopologyName] = useState<string>("");
  const [triggerDraw, setTriggerDraw] = useState(false);

  const { ref } = useResizeObserver<HTMLDivElement>({
    onResize: ({ width, height }) => {
      setFlowDivSize({ height: height ?? 0, width: width ?? 0 });
    },
  });

  return (
    <div className="flex flex-col">
      <VisualizeHeader
        namespace={namespace}
        topologyName={topologyName}
        setNamespace={setNamespace}
        setTopologyName={setTopologyName}
        setTriggerDraw={setTriggerDraw}
      />
      <div
        className="w-[90vw] h-[70vh] p-4"
        ref={ref}
      >
        <div style={{ height: flowDivSize.height, width: flowDivSize.width }}>
          <ReactFlowProvider>
            <VisualizeFlow
              namespace={namespace}
              topologyName={topologyName}
              setTriggerDraw={setTriggerDraw}
              triggerDraw={triggerDraw}
            />
          </ReactFlowProvider>
        </div>
      </div>
    </div>
  );
}
