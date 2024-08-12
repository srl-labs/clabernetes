import { Handle, type NodeProps } from "@xyflow/react";
import type { ReactElement } from "react";
import {
  getBannerColor,
  getHandlePosition,
  getSubBannerColor,
  getSubSubBannerColor,
  HandleType,
} from "@/components/visualizer/node-common.tsx";
import type { LayoutStyle } from "@/components/visualizer/flow.tsx";

interface NodeServiceProps extends NodeProps {
  readonly layoutStyle: LayoutStyle;
}

export function NodeService(props: NodeServiceProps): ReactElement {
  const { data, layoutStyle } = props;

  const kind = "service";
  const serviceKind = data.serviceKind as string;
  const qualifiedKind = `${kind}-${serviceKind}`;
  const name = data.label as string;
  const resourceName = data.resourceName as string;

  return (
    <div className="rounded-md bg-gray-400 text-center shadow-xl">
      <Handle
        className="h-2 w-2"
        position={getHandlePosition(layoutStyle, HandleType.Target)}
        type="target"
      />
      <p className={`rounded-t-md ${getBannerColor(qualifiedKind)} py-1 text-sm text-white`}>
        Service
      </p>
      <p className={`${getSubBannerColor(qualifiedKind)} text-sm text-gray-700`}>{serviceKind}</p>
      <p className={`${getSubSubBannerColor(qualifiedKind)} text-sm text-gray-700`}>
        {resourceName}
      </p>
      <p className="text-sm text-gray-700">{name}</p>
      <Handle
        className="h-2 w-2"
        position={getHandlePosition(layoutStyle, HandleType.Source)}
        type="source"
      />
    </div>
  );
}
