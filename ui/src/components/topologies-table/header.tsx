import { Input } from "@/components/ui/input";
import type { Column } from "@tanstack/react-table";
import type { ReactElement } from "react";
import type { ClabernetesContainerlabDevTopologyV1Alpha1 } from "@/lib/clabernetes-client";

interface HeaderProps {
  readonly getColumn: (
    columnId: string,
  ) => Column<ClabernetesContainerlabDevTopologyV1Alpha1> | undefined;
}

export function Header(props: HeaderProps): ReactElement {
  const { getColumn } = props;

  return (
    <div>
      <div className="flex" />
      <div className="flex justify-center space-x-2 py-4">
        <Input
          className="max-w-sm"
          onChange={(event) => {
            getColumn("namespace")?.setFilterValue(event.target.value);
          }}
          placeholder="Filter by namespace..."
          value={getColumn("namespace")?.getFilterValue() as string}
        />
        <Input
          className="max-w-sm"
          onChange={(event) => {
            getColumn("name")?.setFilterValue(event.target.value);
          }}
          placeholder="Filter by name..."
          value={getColumn("name")?.getFilterValue() as string}
        />
      </div>
    </div>
  );
}
