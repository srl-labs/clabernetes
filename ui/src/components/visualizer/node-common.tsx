import { Position } from "@xyflow/react";
import { Expand, Shrink } from "lucide-react";
import type { ReactElement } from "react";
import { LayoutStyle } from "@/components/visualizer/flow.tsx";

export enum HandleType {
  Target = "target",
  Source = "source",
}

export function getHandlePosition(layoutStyle: LayoutStyle, handleType: HandleType): Position {
  switch (layoutStyle) {
    case LayoutStyle.Horizontal:
      switch (handleType) {
        case HandleType.Source:
          return Position.Right;
        default:
          return Position.Left;
      }
    default:
      switch (handleType) {
        case HandleType.Source:
          return Position.Bottom;
        default:
          return Position.Top;
      }
  }
}

export function getExpandIcon(isOpen: boolean): ReactElement {
  if (isOpen) {
    return <Shrink className="h-4 w-4 fill-current text-black" />;
  }

  return <Expand className="h-4 w-4 fill-current text-black" />;
}

export function getBannerColor(kind: string): string {
  switch (kind) {
    case "topology":
      return "bg-green-900";
    case "deployment":
      return "bg-blue-900";
    case "service-fabric":
      return "bg-purple-900";
    case "service-expose":
      return "bg-pink-900";
    default:
      return "bg-gray-900";
  }
}

export function getSubBannerColor(kind: string): string {
  switch (kind) {
    case "topology":
      return "bg-green-500";
    case "deployment":
      return "bg-blue-500";
    case "service-fabric":
      return "bg-purple-500";
    case "service-expose":
      return "bg-pink-500";
    default:
      return "bg-gray-500";
  }
}

export function getSubSubBannerColor(kind: string): string {
  switch (kind) {
    case "topology":
      return "bg-green-200";
    case "deployment":
      return "bg-blue-200";
    case "service-fabric":
      return "bg-purple-200";
    case "service-expose":
      return "bg-pink-200";
    default:
      return "bg-gray-200";
  }
}
