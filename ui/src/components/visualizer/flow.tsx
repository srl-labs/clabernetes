"use client";
import {
  Background,
  type ColorMode,
  Controls,
  type Edge,
  type NodeProps,
  type NodeTypes,
  Panel,
  ReactFlow,
  useEdgesState,
  useNodesState,
  useReactFlow,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Button } from "@/components/ui/button.tsx";
import { useTheme } from "next-themes";
import { type ReactElement, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Elk, { type LayoutOptions } from "elkjs";
import { type VisualizeObject, visualizeTopology } from "@/lib/kubernetes-visualize.ts";
import { NodeTopology } from "@/components/visualizer/node-topology.tsx";
import { NodeDeployment } from "@/components/visualizer/node-deployment.tsx";
import { NodeService } from "@/components/visualizer/node-service.tsx";
import { NodeInterface } from "@/components/visualizer/node-interface.tsx";

const topologyPartition = 10;
const deploymentPartition = 20;
const servicePartition = 30;
const loadBalancerPartition = 40;
const interfacePartition = 50;

const redrawDelay = 25;

export enum VisualizeStyle {
  Kubernetes = "kubernetes",
  Network = "network",
}

const elk = new Elk({});

export enum LayoutStyle {
  Vertical = "vertical",
  Horizontal = "right",
}

interface VisualizeObjectElk {
  data: Record<string, unknown>;
  height: number;
  id: string;
  labels: Record<string, string>[];
  layoutOptions: Record<string, string | number>;
  type: string;
  width: number;
  x: number;
  y: number;
}

function getElkPartitionId(obj: VisualizeObject): number {
  switch (obj.data.kind) {
    case "deployment":
      return deploymentPartition;
    case "service":
      return servicePartition;
    case "loadBalancer":
      return loadBalancerPartition;
    case "interface":
      return interfacePartition;
    default:
      return topologyPartition;
  }
}

function visualizeObjectToVisualizeObjectElk(obj: VisualizeObject): VisualizeObjectElk {
  return {
    data: obj.data,
    height: obj.style.height,
    id: obj.id,
    // exists for making it easier to look at in elk online json editor
    labels: [{ text: obj.id }],
    layoutOptions: {
      "partitioning.partition": getElkPartitionId(obj),
    },
    type: obj.type,
    width: obj.style.width,
    x: obj.position.x,
    y: obj.position.y,
  };
}

function visualizeObjectElkToVisualizeObject(obj: VisualizeObjectElk): VisualizeObject {
  return {
    data: obj.data,
    id: obj.id,
    position: {
      x: obj.x,
      y: obj.y,
    },
    style: {
      height: obj.height,
      width: obj.width,
    },
    type: obj.type,
  };
}

function getElkOptions(style: LayoutStyle): LayoutOptions {
  const baseOptions = {
    "elk.layered.nodePlacement.strategy": "NETWORK_SIMPLEX",
    "elk.layered.spacing.edgeNodeBetweenLayers": "100",
    "elk.layered.spacing.nodeNodeBetweenLayers": "100",
    "elk.partitioning.activate": "true",
    "elk.separateConnectedComponents": "true",
    "elk.spacing.componentComponent": "100",
    "elk.spacing.nodeNode": "100",
  };

  switch (style) {
    case LayoutStyle.Vertical: {
      return {
        ...baseOptions,
        "elk.algorithm": "layered",
        "elk.direction": "DOWN",
      };
    }
    default: {
      return {
        ...baseOptions,
        "elk.algorithm": "layered",
        "elk.direction": "RIGHT",
      };
    }
  }
}

function visualizeObjectsToVisualizeObjectsElk(
  initialNodes: VisualizeObject[],
): VisualizeObjectElk[] {
  const computedNodes: VisualizeObjectElk[] = [];

  for (const initialNode of initialNodes) {
    computedNodes.push(visualizeObjectToVisualizeObjectElk(initialNode));
  }

  return computedNodes;
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: its fiiiiiine
function kubernetesToNetworkNodesAndEdges(
  initialNodes: VisualizeObject[],
  initialEdges: Edge[],
): [VisualizeObject[], Edge[]] {
  const networkNodes: VisualizeObject[] = [];
  const networkEdges: Edge[] = [];

  for (const initialNode of initialNodes) {
    switch (initialNode.type) {
      case "topology":
        // we remove the edges attaching to the topo and we *dont* put the topo node in the
        // array of nodes to draw
        networkEdges.forEach((edge, index) => {
          if (edge.source === initialNode.id) {
            networkEdges.splice(index, 1);
          }
        });

        break;
      case "service":
        switch (initialNode.data.serviceKind) {
          case "expose":
            // expose service is like topology, just get rid of it and its edges
            break;
          default: {
            let upstreamDeployment = "";

            for (const edge of initialEdges) {
              if (edge.target === initialNode.id) {
                // services only have the one upstream deployment, so this is that
                upstreamDeployment = edge.source;

                break;
              }
            }

            for (const edge of initialEdges) {
              if (edge.source === initialNode.id) {
                // we found a service -> interface edge, lets replace it with a deployment ->
                // interface edge
                networkEdges.push({
                  id: `${upstreamDeployment} / ${edge.target}`,
                  source: upstreamDeployment,
                  target: edge.target,
                });

                for (const innerEdge of initialEdges) {
                  if (edge.target === innerEdge.source) {
                    networkEdges.push({
                      id: `${innerEdge.source} / ${edge.target}`,
                      source: edge.target,
                      target: innerEdge.target,
                    });
                  }
                }
              }
            }

            break;
          }
        }
        break;
      default:
        networkNodes.push(initialNode);

        break;
    }
  }

  return [networkNodes, networkEdges];
}

async function createLayout(
  initialNodes: VisualizeObject[],
  initialEdges: Edge[],
  layoutStyle: LayoutStyle,
  visualizeStyle: VisualizeStyle,
): Promise<{
  nodes: VisualizeObject[];
  edges: Edge[];
}> {
  if (visualizeStyle === VisualizeStyle.Network) {
    // biome-ignore lint/style/noParameterAssign: blah
    [initialNodes, initialEdges] = kubernetesToNetworkNodesAndEdges(initialNodes, initialEdges);
  }

  const graph = {
    children: [],
    edges: [],
    id: "root",
    layoutOptions: getElkOptions(layoutStyle),
  };

  const computedNodes = visualizeObjectsToVisualizeObjectsElk(initialNodes);

  // @ts-expect-error elk not typed how i want
  graph.children = computedNodes;

  for (const initialEdge of initialEdges) {
    // @ts-expect-error elk not typed how i want it to be
    graph.edges.push({
      id: initialEdge.id,
      sources: [initialEdge.source],
      targets: [initialEdge.target],
    });
  }

  const layout = await elk.layout(graph);

  const layoutedNodes =
    layout.children?.map((obj) => {
      // @ts-expect-error elk not typed how i want
      return visualizeObjectElkToVisualizeObject(obj);
    }) ?? [];

  return {
    nodes: layoutedNodes,
    edges: initialEdges,
  };
}

function getFlowTheme(resolvedTheme: string | undefined): ColorMode {
  if (resolvedTheme === "light") {
    return "light";
  }

  return "dark";
}

interface VisualizeFlowProps {
  readonly namespace: string;
  readonly topologyName: string;
  readonly setTriggerDraw: (state: boolean) => void;
  readonly triggerDraw: boolean;
}

export function VisualizeFlow(props: VisualizeFlowProps): ReactElement {
  const { namespace, topologyName, setTriggerDraw, triggerDraw } = props;

  const theme = useTheme();

  const reactFlow = useReactFlow();

  const [visualizeStyle, setVisualizeStyle] = useState<VisualizeStyle>(VisualizeStyle.Kubernetes);
  const [layoutStyle, setLayoutStyle] = useState<LayoutStyle>(LayoutStyle.Horizontal);
  const [nodes, setNodes, onNodesChange] = useNodesState<VisualizeObject>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  const { data } = useQuery({
    enabled: namespace !== "" && topologyName !== "",
    queryFn: async (): Promise<{ nodes: VisualizeObject[]; edges: Edge[] }> => {
      const response = await visualizeTopology(namespace, topologyName);

      return JSON.parse(response);
    },
    queryKey: ["visualize", { namespace: namespace, topologyName: topologyName }],
    refetchIntervalInBackground: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    retry: false,
    throwOnError: true,
  });

  useEffect(() => {
    if (!triggerDraw) {
      return;
    }

    if (!data || Object.keys(data.nodes).length === 0) {
      setNodes([]);
      setEdges([]);
      setTriggerDraw(false);
      setTimeout(reactFlow.fitView, redrawDelay);
      return;
    }

    createLayout(data.nodes, data.edges, layoutStyle, visualizeStyle)
      .catch((layoutErr: unknown) => {
        throw layoutErr;
      })
      .then(({ nodes: layoutedNodes, edges: layoutedEdges }) => {
        setNodes(layoutedNodes);
        setEdges(layoutedEdges);
        setTimeout(reactFlow.fitView, redrawDelay);
      });

    setTriggerDraw(false);
  }, [
    data,
    layoutStyle,
    visualizeStyle,
    reactFlow.fitView,
    setEdges,
    setNodes,
    triggerDraw,
    setTriggerDraw,
  ]);

  const nodeTypes = useMemo((): NodeTypes => {
    return {
      deployment: (nodeProps: NodeProps): ReactElement => {
        return (
          <NodeDeployment
            layoutStyle={layoutStyle}
            {...nodeProps}
          />
        );
      },
      interface: (nodeProps: NodeProps): ReactElement => {
        return (
          <NodeInterface
            layoutStyle={layoutStyle}
            {...nodeProps}
          />
        );
      },
      service: (nodeProps: NodeProps): ReactElement => {
        return (
          <NodeService
            layoutStyle={layoutStyle}
            {...nodeProps}
          />
        );
      },
      topology: (nodeProps: NodeProps): ReactElement => {
        return (
          <NodeTopology
            layoutStyle={layoutStyle}
            {...nodeProps}
          />
        );
      },
    };
  }, [layoutStyle]);

  return (
    <ReactFlow
      colorMode={getFlowTheme(theme.resolvedTheme)}
      edges={edges}
      fitView={true}
      maxZoom={2}
      minZoom={0.1}
      nodes={nodes}
      nodeTypes={nodeTypes}
      onEdgesChange={onEdgesChange}
      onNodesChange={onNodesChange}
    >
      <Panel
        className="space-x-2 p-2"
        position="top-left"
      >
        <Button
          disabled={namespace === "" || layoutStyle === LayoutStyle.Horizontal}
          onClick={(): void => {
            setLayoutStyle(LayoutStyle.Horizontal);
            setTriggerDraw(true);
          }}
          variant="secondary"
        >
          Horizontal
        </Button>
        <Button
          disabled={namespace === "" || layoutStyle === LayoutStyle.Vertical}
          onClick={(): void => {
            setLayoutStyle(LayoutStyle.Vertical);
            setTriggerDraw(true);
          }}
          variant="secondary"
        >
          Vertical
        </Button>
      </Panel>
      <Panel
        className="space-x-2 p-2"
        position="top-right"
      >
        <Button
          disabled={namespace === "" || visualizeStyle === VisualizeStyle.Kubernetes}
          onClick={(): void => {
            setVisualizeStyle(VisualizeStyle.Kubernetes);
            setTriggerDraw(true);
          }}
          variant="secondary"
        >
          Kubernetes
        </Button>
        <Button
          disabled={namespace === "" || visualizeStyle === VisualizeStyle.Network}
          onClick={(): void => {
            setVisualizeStyle(VisualizeStyle.Network);
            setTriggerDraw(true);
          }}
          variant="secondary"
        >
          Network
        </Button>
      </Panel>
      <Background />
      <Controls />
    </ReactFlow>
  );
}
