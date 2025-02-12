// This file is auto-generated by @hey-api/openapi-ts

import type { Options as ClientOptions, TDataShape, Client } from '@hey-api/client-fetch';
import type { ListClabernetesContainerlabDevV1Alpha1ConnectivityForAllNamespacesData, ListClabernetesContainerlabDevV1Alpha1ConnectivityForAllNamespacesResponse, DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConnectivityData, DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConnectivityResponse, ListClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, ListClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, CreateClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, CreateClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, DeleteClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, DeleteClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, ReadClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, ReadClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, PatchClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, PatchClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, ReplaceClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, ReplaceClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, ListClabernetesContainerlabDevV1Alpha1ImagerequestForAllNamespacesData, ListClabernetesContainerlabDevV1Alpha1ImagerequestForAllNamespacesResponse, DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedImagerequestData, DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedImagerequestResponse, ListClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, ListClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, CreateClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, CreateClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, DeleteClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, DeleteClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, ReadClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, ReadClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, PatchClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, PatchClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, ReplaceClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, ReplaceClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, ListClabernetesContainerlabDevV1Alpha1ConfigForAllNamespacesData, ListClabernetesContainerlabDevV1Alpha1ConfigForAllNamespacesResponse, DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConfigData, DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConfigResponse, ListClabernetesContainerlabDevV1Alpha1NamespacedConfigData, ListClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, CreateClabernetesContainerlabDevV1Alpha1NamespacedConfigData, CreateClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, DeleteClabernetesContainerlabDevV1Alpha1NamespacedConfigData, DeleteClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, ReadClabernetesContainerlabDevV1Alpha1NamespacedConfigData, ReadClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, PatchClabernetesContainerlabDevV1Alpha1NamespacedConfigData, PatchClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, ReplaceClabernetesContainerlabDevV1Alpha1NamespacedConfigData, ReplaceClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, ListClabernetesContainerlabDevV1Alpha1TopologyForAllNamespacesData, ListClabernetesContainerlabDevV1Alpha1TopologyForAllNamespacesResponse, DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedTopologyData, DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedTopologyResponse, ListClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, ListClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, CreateClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, CreateClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, DeleteClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, DeleteClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, ReadClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, ReadClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, PatchClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, PatchClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, ReplaceClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, ReplaceClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse } from './types.gen';
import { client as _heyApiClient } from './client.gen';

export type Options<TData extends TDataShape = TDataShape, ThrowOnError extends boolean = boolean> = ClientOptions<TData, ThrowOnError> & {
    /**
     * You can provide a client instance returned by `createClient()` instead of
     * individual options. This might be also useful if you want to implement a
     * custom client.
     */
    client?: Client;
    /**
     * You can pass arbitrary values through the `meta` object. This can be
     * used to access values that aren't defined as part of the SDK function.
     */
    meta?: Record<string, unknown>;
};

/**
 * list objects of kind Connectivity
 */
export const listClabernetesContainerlabDevV1Alpha1ConnectivityForAllNamespaces = <ThrowOnError extends boolean = false>(options?: Options<ListClabernetesContainerlabDevV1Alpha1ConnectivityForAllNamespacesData, ThrowOnError>) => {
    return (options?.client ?? _heyApiClient).get<ListClabernetesContainerlabDevV1Alpha1ConnectivityForAllNamespacesResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/connectivities',
        ...options
    });
};

/**
 * delete collection of Connectivity
 */
export const deleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConnectivity = <ThrowOnError extends boolean = false>(options: Options<DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConnectivityData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).delete<DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConnectivityResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/connectivities',
        ...options
    });
};

/**
 * list objects of kind Connectivity
 */
export const listClabernetesContainerlabDevV1Alpha1NamespacedConnectivity = <ThrowOnError extends boolean = false>(options: Options<ListClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).get<ListClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/connectivities',
        ...options
    });
};

/**
 * create a Connectivity
 */
export const createClabernetesContainerlabDevV1Alpha1NamespacedConnectivity = <ThrowOnError extends boolean = false>(options: Options<CreateClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).post<CreateClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/connectivities',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * delete a Connectivity
 */
export const deleteClabernetesContainerlabDevV1Alpha1NamespacedConnectivity = <ThrowOnError extends boolean = false>(options: Options<DeleteClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).delete<DeleteClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/connectivities/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * read the specified Connectivity
 */
export const readClabernetesContainerlabDevV1Alpha1NamespacedConnectivity = <ThrowOnError extends boolean = false>(options: Options<ReadClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).get<ReadClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/connectivities/{name}',
        ...options
    });
};

/**
 * partially update the specified Connectivity
 */
export const patchClabernetesContainerlabDevV1Alpha1NamespacedConnectivity = <ThrowOnError extends boolean = false>(options: Options<PatchClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).patch<PatchClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/connectivities/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/apply-patch+yaml',
            ...options?.headers
        }
    });
};

/**
 * replace the specified Connectivity
 */
export const replaceClabernetesContainerlabDevV1Alpha1NamespacedConnectivity = <ThrowOnError extends boolean = false>(options: Options<ReplaceClabernetesContainerlabDevV1Alpha1NamespacedConnectivityData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).put<ReplaceClabernetesContainerlabDevV1Alpha1NamespacedConnectivityResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/connectivities/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * list objects of kind Imagerequest
 */
export const listClabernetesContainerlabDevV1Alpha1ImagerequestForAllNamespaces = <ThrowOnError extends boolean = false>(options?: Options<ListClabernetesContainerlabDevV1Alpha1ImagerequestForAllNamespacesData, ThrowOnError>) => {
    return (options?.client ?? _heyApiClient).get<ListClabernetesContainerlabDevV1Alpha1ImagerequestForAllNamespacesResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/imagerequests',
        ...options
    });
};

/**
 * delete collection of Imagerequest
 */
export const deleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedImagerequest = <ThrowOnError extends boolean = false>(options: Options<DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedImagerequestData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).delete<DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedImagerequestResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/imagerequests',
        ...options
    });
};

/**
 * list objects of kind Imagerequest
 */
export const listClabernetesContainerlabDevV1Alpha1NamespacedImagerequest = <ThrowOnError extends boolean = false>(options: Options<ListClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).get<ListClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/imagerequests',
        ...options
    });
};

/**
 * create a Imagerequest
 */
export const createClabernetesContainerlabDevV1Alpha1NamespacedImagerequest = <ThrowOnError extends boolean = false>(options: Options<CreateClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).post<CreateClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/imagerequests',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * delete a Imagerequest
 */
export const deleteClabernetesContainerlabDevV1Alpha1NamespacedImagerequest = <ThrowOnError extends boolean = false>(options: Options<DeleteClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).delete<DeleteClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/imagerequests/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * read the specified Imagerequest
 */
export const readClabernetesContainerlabDevV1Alpha1NamespacedImagerequest = <ThrowOnError extends boolean = false>(options: Options<ReadClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).get<ReadClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/imagerequests/{name}',
        ...options
    });
};

/**
 * partially update the specified Imagerequest
 */
export const patchClabernetesContainerlabDevV1Alpha1NamespacedImagerequest = <ThrowOnError extends boolean = false>(options: Options<PatchClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).patch<PatchClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/imagerequests/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/apply-patch+yaml',
            ...options?.headers
        }
    });
};

/**
 * replace the specified Imagerequest
 */
export const replaceClabernetesContainerlabDevV1Alpha1NamespacedImagerequest = <ThrowOnError extends boolean = false>(options: Options<ReplaceClabernetesContainerlabDevV1Alpha1NamespacedImagerequestData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).put<ReplaceClabernetesContainerlabDevV1Alpha1NamespacedImagerequestResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/imagerequests/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * list objects of kind Config
 */
export const listClabernetesContainerlabDevV1Alpha1ConfigForAllNamespaces = <ThrowOnError extends boolean = false>(options?: Options<ListClabernetesContainerlabDevV1Alpha1ConfigForAllNamespacesData, ThrowOnError>) => {
    return (options?.client ?? _heyApiClient).get<ListClabernetesContainerlabDevV1Alpha1ConfigForAllNamespacesResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/configs',
        ...options
    });
};

/**
 * delete collection of Config
 */
export const deleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConfig = <ThrowOnError extends boolean = false>(options: Options<DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConfigData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).delete<DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedConfigResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/configs',
        ...options
    });
};

/**
 * list objects of kind Config
 */
export const listClabernetesContainerlabDevV1Alpha1NamespacedConfig = <ThrowOnError extends boolean = false>(options: Options<ListClabernetesContainerlabDevV1Alpha1NamespacedConfigData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).get<ListClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/configs',
        ...options
    });
};

/**
 * create a Config
 */
export const createClabernetesContainerlabDevV1Alpha1NamespacedConfig = <ThrowOnError extends boolean = false>(options: Options<CreateClabernetesContainerlabDevV1Alpha1NamespacedConfigData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).post<CreateClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/configs',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * delete a Config
 */
export const deleteClabernetesContainerlabDevV1Alpha1NamespacedConfig = <ThrowOnError extends boolean = false>(options: Options<DeleteClabernetesContainerlabDevV1Alpha1NamespacedConfigData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).delete<DeleteClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/configs/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * read the specified Config
 */
export const readClabernetesContainerlabDevV1Alpha1NamespacedConfig = <ThrowOnError extends boolean = false>(options: Options<ReadClabernetesContainerlabDevV1Alpha1NamespacedConfigData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).get<ReadClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/configs/{name}',
        ...options
    });
};

/**
 * partially update the specified Config
 */
export const patchClabernetesContainerlabDevV1Alpha1NamespacedConfig = <ThrowOnError extends boolean = false>(options: Options<PatchClabernetesContainerlabDevV1Alpha1NamespacedConfigData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).patch<PatchClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/configs/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/apply-patch+yaml',
            ...options?.headers
        }
    });
};

/**
 * replace the specified Config
 */
export const replaceClabernetesContainerlabDevV1Alpha1NamespacedConfig = <ThrowOnError extends boolean = false>(options: Options<ReplaceClabernetesContainerlabDevV1Alpha1NamespacedConfigData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).put<ReplaceClabernetesContainerlabDevV1Alpha1NamespacedConfigResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/configs/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * list objects of kind Topology
 */
export const listClabernetesContainerlabDevV1Alpha1TopologyForAllNamespaces = <ThrowOnError extends boolean = false>(options?: Options<ListClabernetesContainerlabDevV1Alpha1TopologyForAllNamespacesData, ThrowOnError>) => {
    return (options?.client ?? _heyApiClient).get<ListClabernetesContainerlabDevV1Alpha1TopologyForAllNamespacesResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/topologies',
        ...options
    });
};

/**
 * delete collection of Topology
 */
export const deleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedTopology = <ThrowOnError extends boolean = false>(options: Options<DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedTopologyData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).delete<DeleteClabernetesContainerlabDevV1Alpha1CollectionNamespacedTopologyResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/topologies',
        ...options
    });
};

/**
 * list objects of kind Topology
 */
export const listClabernetesContainerlabDevV1Alpha1NamespacedTopology = <ThrowOnError extends boolean = false>(options: Options<ListClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).get<ListClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/topologies',
        ...options
    });
};

/**
 * create a Topology
 */
export const createClabernetesContainerlabDevV1Alpha1NamespacedTopology = <ThrowOnError extends boolean = false>(options: Options<CreateClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).post<CreateClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/topologies',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * delete a Topology
 */
export const deleteClabernetesContainerlabDevV1Alpha1NamespacedTopology = <ThrowOnError extends boolean = false>(options: Options<DeleteClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).delete<DeleteClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/topologies/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};

/**
 * read the specified Topology
 */
export const readClabernetesContainerlabDevV1Alpha1NamespacedTopology = <ThrowOnError extends boolean = false>(options: Options<ReadClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).get<ReadClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/topologies/{name}',
        ...options
    });
};

/**
 * partially update the specified Topology
 */
export const patchClabernetesContainerlabDevV1Alpha1NamespacedTopology = <ThrowOnError extends boolean = false>(options: Options<PatchClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).patch<PatchClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/topologies/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/apply-patch+yaml',
            ...options?.headers
        }
    });
};

/**
 * replace the specified Topology
 */
export const replaceClabernetesContainerlabDevV1Alpha1NamespacedTopology = <ThrowOnError extends boolean = false>(options: Options<ReplaceClabernetesContainerlabDevV1Alpha1NamespacedTopologyData, ThrowOnError>) => {
    return (options.client ?? _heyApiClient).put<ReplaceClabernetesContainerlabDevV1Alpha1NamespacedTopologyResponse, unknown, ThrowOnError>({
        url: '/apis/clabernetes.containerlab.dev/v1alpha1/namespaces/{namespace}/topologies/{name}',
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options?.headers
        }
    });
};