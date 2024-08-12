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

interface NodeDeploymentProps extends NodeProps {
  readonly layoutStyle: LayoutStyle;
}

export function NodeDeployment(props: NodeDeploymentProps): ReactElement {
  const { data, layoutStyle } = props;

  const kind = "deployment";
  const name = data.label as string;
  const resourceName = data.resourceName as string;

  return (
    <div className="rounded-md bg-gray-400 text-center shadow-xl">
      <Handle
        className="h-2 w-2"
        position={getHandlePosition(layoutStyle, HandleType.Target)}
        type="target"
      />
      <p className={`rounded-t-md ${getBannerColor(kind)} py-1 text-sm text-white`}>Deployment</p>
      <p className={`${getSubBannerColor(kind)} text-sm text-gray-700`}>{resourceName}</p>
      <p className={`${getSubSubBannerColor(kind)} text-sm text-gray-700 min-h-10 rounded-md`}>
        {name}
      </p>
      <Handle
        className="h-2 w-2"
        position={getHandlePosition(layoutStyle, HandleType.Source)}
        type="source"
      />
    </div>
  );
}
