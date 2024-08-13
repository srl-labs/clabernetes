import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { ReactElement } from "react";
import type { ClabernetesContainerlabDevTopologyV1Alpha1 } from "@/lib/clabernetes-client";
import type { Row } from "@tanstack/react-table";
import { CircleAlert, CircleCheck, CircleHelp } from "lucide-react";

const kindPattern = /kind: (.*)/;
const imagePattern = /image: (.*)/;

function getTopologyReadyIcon(topologyReady: boolean | undefined): ReactElement {
  switch (topologyReady) {
    case undefined:
      return <CircleHelp className="h-4 w-4 mt-1 fill-yellow-500" />;
    case true:
      return <CircleCheck className="h-4 w-4 mt-1 fill-green-500" />;
    default:
      return <CircleAlert className="h-4 w-4 mt-1 fill-red-500" />;
  }
}

function getMatchOrUnknown(text: string, pattern: RegExp): string {
  const match = text.match(pattern);

  if (match === null) {
    return "unknown";
  }

  if (match.length !== 2) {
    return "unknown";
  }

  return match[1];
}

function getTopologyNodeCard(
  nodeName: string,
  obj: ClabernetesContainerlabDevTopologyV1Alpha1,
): ReactElement {
  const nodeConfig = obj.status?.configs[nodeName] ?? "";
  const nodeExposedPortData = obj.status?.exposedPorts[nodeName];

  const nodeReadiness = obj.status?.nodeReadiness[nodeName];
  const kind = getMatchOrUnknown(nodeConfig, kindPattern);
  const image = getMatchOrUnknown(nodeConfig, imagePattern);
  const loadBalancerAddress = nodeExposedPortData?.loadBalancerAddress;
  const exposedTcpPorts = nodeExposedPortData?.tcpPorts ?? [];
  const exposedUdpPorts = nodeExposedPortData?.udpPorts ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-center">{nodeName}</CardTitle>
      </CardHeader>
      <CardContent className="flex items-center justify-center">
        <div className="flex flex-col text-sm font-normal">
          <div className="flex items-center">
            <span className="w-24 pr-2 text-right font-semibold">Readiness:</span>
            <span>{nodeReadiness}</span>
          </div>
          <div className="flex items-center">
            <span className="w-24 pr-2 text-right font-semibold">Kind:</span>
            <span>{kind}</span>
          </div>
          <div className="flex items-center">
            <span className="w-24 pr-2 text-right font-semibold">Image:</span>
            <span>{`${image}`}</span>
          </div>
          <div className="flex items-center">
            <span className="w-24 pr-2 text-right font-semibold">LB Address:</span>
            <span>{loadBalancerAddress}</span>
          </div>
          <div className="flex items-center">
            <span className="w-24 pr-2 text-right font-semibold">TCP Ports:</span>
          </div>
          <ul className="pl-24 list-disc">
            {exposedTcpPorts.map((port, index) => (
              <li key={`${index}-${port}`}>{port}</li>
            ))}
          </ul>
          <div className="flex items-center">
            <span className="w-24 pr-2 text-right font-semibold">UDP Ports:</span>
          </div>
          <ul className="pl-24 list-disc">
            {exposedUdpPorts.map((port, index) => (
              <li key={`${index}-${port}`}>{port}</li>
            ))}
          </ul>
        </div>
      </CardContent>
    </Card>
  );
}

interface ExpandProps {
  readonly row: Row<ClabernetesContainerlabDevTopologyV1Alpha1>;
}

export function Expand(props: ExpandProps): ReactElement {
  const { row } = props;

  const obj = row.original;
  const objNodes = obj.status?.configs ? Array.from(Object.keys(obj.status?.configs)) : [];

  return (
    <div>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-center">
            <div className="flex flex-col text-sm font-normal">
              <div className="flex items-center">
                <span className="w-24 pr-2 text-right font-semibold">Namespace:</span>
                <span>{obj.metadata?.namespace as string}</span>
              </div>
              <div className="flex items-center">
                <span className="w-24 pr-2 text-right font-semibold">Name:</span>
                <span>{obj.metadata?.name as string}</span>
              </div>
              <div className="flex items-center">
                <span className="w-24 pr-2 text-right font-semibold">Ready:</span>
                <span>{getTopologyReadyIcon(obj.status?.topologyReady)}</span>
              </div>
            </div>
          </CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
          {objNodes.map((nodeName) => {
            return getTopologyNodeCard(nodeName, obj);
          })}
        </CardContent>
      </Card>
    </div>
  );
}
