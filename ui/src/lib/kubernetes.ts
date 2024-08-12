"use server";
import "../fetch.config";
import { CoreV1Api, KubeConfig } from "@kubernetes/client-node";

import {
  deleteClabernetesContainerlabDevV1Alpha1NamespacedTopology,
  listClabernetesContainerlabDevV1Alpha1NamespacedTopology,
  listClabernetesContainerlabDevV1Alpha1TopologyForAllNamespaces,
  replaceClabernetesContainerlabDevV1Alpha1NamespacedTopology,
} from "@/lib/clabernetes-client";

export async function listTopologies(): Promise<string> {
  const response = await listClabernetesContainerlabDevV1Alpha1TopologyForAllNamespaces().catch(
    (error: unknown) => {
      throw error;
    },
  );

  return JSON.stringify(response.data?.items);
}

export async function listNamespacedTopologies(namespace: string): Promise<string> {
  const response = await listClabernetesContainerlabDevV1Alpha1NamespacedTopology({
    path: { namespace: namespace },
  }).catch((error: unknown) => {
    throw error;
  });

  return JSON.stringify(
    response.data?.items.map((namespace) => {
      return namespace.metadata?.name;
    }),
  );
}

export async function deleteTopology(namespace: string, name: string): Promise<string> {
  const response = await deleteClabernetesContainerlabDevV1Alpha1NamespacedTopology({
    path: { name: name, namespace: namespace },
  });

  return JSON.stringify(response);
}

export async function updateTopology(
  namespace: string,
  name: string,
  body: string,
): Promise<string> {
  const response = await replaceClabernetesContainerlabDevV1Alpha1NamespacedTopology({
    body: JSON.parse(body),
    path: { name: name, namespace: namespace },
  });

  return JSON.stringify(response);
}

export async function listNamespaces(): Promise<string> {
  const kc = new KubeConfig();

  kc.loadFromDefault();

  const response = await kc
    .makeApiClient(CoreV1Api)
    .listNamespace()
    .catch((error: unknown) => {
      throw error;
    });

  return JSON.stringify(
    response.body.items.map((namespace) => {
      return namespace.metadata?.name;
    }),
  );
}
