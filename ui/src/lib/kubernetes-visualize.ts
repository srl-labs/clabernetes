"use server";
import type { Edge } from "@xyflow/react";
import {
  AppsV1Api,
  CoreV1Api,
  KubeConfig,
  type V1DeploymentList,
  type V1ServiceList,
} from "@kubernetes/client-node";
import { readClabernetesContainerlabDevV1Alpha1NamespacedConnectivity } from "@/lib/clabernetes-client";

async function deploymentsByOwner(
  namespace: string,
  owningTopologyName: string,
): Promise<V1DeploymentList> {
  const labelSelector = `clabernetes/topologyOwner=${owningTopologyName}`;
  const kc = new KubeConfig();

  kc.loadFromDefault();

  const response = await kc
    .makeApiClient(AppsV1Api)
    .listNamespacedDeployment(namespace, undefined, undefined, undefined, undefined, labelSelector)
    .catch((error: unknown) => {
      throw error;
    });

  return response.body;
}

async function servicesByOwner(
  namespace: string,
  owningTopologyName: string,
): Promise<V1ServiceList> {
  const labelSelector = `clabernetes/topologyOwner=${owningTopologyName}`;
  const kc = new KubeConfig();

  kc.loadFromDefault();

  const response = await kc
    .makeApiClient(CoreV1Api)
    .listNamespacedService(namespace, undefined, undefined, undefined, undefined, labelSelector)
    .catch((error: unknown) => {
      throw error;
    });

  return response.body;
}

export interface VisualizeObject {
  data: Record<string, unknown>;
  id: string;
  type: string;
  position: {
    x: number;
    y: number;
  };
  style: {
    height: number;
    width: number;
  };
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: its fiiiiiine
export async function visualizeTopology(namespace: string, name: string): Promise<string> {
  const nodes: VisualizeObject[] = [];
  const edges: Edge[] = [];

  const deployments = await deploymentsByOwner(namespace, name);

  const services = await servicesByOwner(namespace, name);

  const connectivity = await readClabernetesContainerlabDevV1Alpha1NamespacedConnectivity({
    path: { name: name, namespace: namespace },
  }).catch((error: unknown) => {
    throw error;
  });

  nodes.push({
    data: {
      label: name,
      resourceName: name,
    },
    id: name,
    position: { x: 0, y: 0 },
    style: { height: 90, width: 150 },
    type: "topology",
  });

  for (const deployment of deployments.items) {
    const labels = deployment.metadata?.labels ?? {};
    const deploymentName = labels["clabernetes/name"] ?? "";
    const containerlabNodeName = labels["clabernetes/topologyNode"] ?? "";

    nodes.push({
      data: {
        label: containerlabNodeName,
        resourceName: deployment.metadata?.name as string,
      },
      id: deploymentName,
      position: { x: 0, y: 0 },
      style: { height: 90, width: 150 },
      type: "deployment",
    });

    edges.push({
      id: `${name} / ${deploymentName}`,
      source: name,
      target: deploymentName,
    });
  }

  for (const service of services.items) {
    const serviceName = service.metadata?.name ?? "";
    const qualifiedServiceName = `svc/${serviceName}`;

    const labels = service.metadata?.labels ?? {};
    const deploymentName = labels["clabernetes/name"] ?? "";
    const containerlabNodeName = labels["clabernetes/topologyNode"] ?? "";
    const serviceType = labels["clabernetes/topologyServiceType"] ?? "";

    nodes.push({
      data: {
        label: `${containerlabNodeName}-${serviceType}`,
        serviceKind: serviceType,
        resourceName: service.metadata?.name as string,
      },
      id: qualifiedServiceName,
      position: { x: 0, y: 0 },
      style: { height: 90, width: 150 },
      type: "service",
    });

    edges.push({
      id: `${deploymentName} / ${qualifiedServiceName}`,
      source: deploymentName,
      target: qualifiedServiceName,
    });
  }

  const recordedTunnels: Record<number, boolean> = {};

  // @ts-ignore-error
  const connectivityNameLabel = connectivity.data?.metadata?.labels["clabernetes/name"];

  // doing this to de-dup things because we have both sides of tunnels represented basically
  for (const tunnelDefinitions of Object.values(
    connectivity.data?.spec?.pointToPointTunnels ?? {},
  )) {
    for (const tunnelDefinition of tunnelDefinitions) {
      if (tunnelDefinition.tunnelID in recordedTunnels) {
        continue;
      }

      recordedTunnels[tunnelDefinition.tunnelID] = true;

      const localFabricService = `svc/${connectivityNameLabel}-${tunnelDefinition.localNode}-vx`;
      const localInterface = `${tunnelDefinition.localNode}-${tunnelDefinition.localInterface}`;
      const remoteFabricService = `svc/${connectivityNameLabel}-${tunnelDefinition.remoteNode}-vx`;
      const remoteInterface = `${tunnelDefinition.remoteNode}-${tunnelDefinition.remoteInterface}`;

      nodes.push({
        data: {
          label: localInterface,
          owningNode: tunnelDefinition.localNode,
        },
        id: localInterface,
        position: { x: 0, y: 0 },
        style: { height: 50, width: 150 },
        type: "interface",
      });

      edges.push({
        id: `${localFabricService} / ${localInterface}`,
        source: localFabricService,
        target: localInterface,
      });

      nodes.push({
        data: {
          label: remoteInterface,
          owningNode: tunnelDefinition.remoteNode,
        },
        id: remoteInterface,
        position: { x: 0, y: 0 },
        style: { height: 50, width: 150 },
        type: "interface",
      });

      edges.push({
        id: `${remoteFabricService} / ${remoteInterface}`,
        source: remoteFabricService,
        target: remoteInterface,
      });

      edges.push({
        id: `${localInterface} / ${remoteInterface}`,
        source: localInterface,
        target: remoteInterface,
      });
    }
  }

  return JSON.stringify({
    edges: edges,
    nodes: nodes,
  });
}
