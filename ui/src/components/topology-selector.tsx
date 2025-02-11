import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { listNamespacedTopologies } from "@/lib/kubernetes";
import { useQuery } from "@tanstack/react-query";
import { ChevronDown } from "lucide-react";
import type { ReactElement } from "react";

interface NamespaceSelectorProps {
  readonly namespace: string;
  readonly topologyName: string;
  readonly placeholder: string;
  readonly setTopologyName: (topologyName: string) => void;
}

function renderDropdownMenuTrigger(namespace: string, placeholder: string): ReactElement {
  if (namespace) {
    return <span>{namespace}</span>;
  }

  if (placeholder) {
    return <span>{placeholder}</span>;
  }

  return <span />;
}

export function TopologySelector(props: NamespaceSelectorProps): ReactElement {
  const { namespace, topologyName, placeholder, setTopologyName } = props;

  const { data, isPending, isError, error } = useQuery<string[], Error>({
    enabled: namespace !== "",
    // biome-ignore lint/suspicious/noExplicitAny: matching queryFn expectations/json.parse return
    queryFn: async (): Promise<any> => {
      const response = await listNamespacedTopologies(namespace);

      return JSON.parse(response);
    },
    queryKey: ["topology", { namespace: namespace }],
  });

  if (isPending) {
    return (
      <div className="flex justify-end col-span-3">
        <Button
          className="flex w-full justify-between"
          variant="outline"
        >
          <span>Loading...</span>
          <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
        </Button>
      </div>
    );
  }

  if (isError) {
    return <span>Error: {error.message}</span>;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        asChild={true}
        className="col-span-3"
      >
        <div className="flex justify-end">
          <Button
            className="flex w-full justify-between"
            variant="outline"
          >
            {renderDropdownMenuTrigger(topologyName, placeholder)}
            <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
          </Button>
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuContent style={{ minWidth: "30vw" }}>
        {data.map((curTopologyName: string) => {
          return (
            <DropdownMenuCheckboxItem
              checked={topologyName === curTopologyName}
              key={curTopologyName}
              onClick={(): void => {
                setTopologyName(curTopologyName);
              }}
            >
              {curTopologyName}
            </DropdownMenuCheckboxItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
