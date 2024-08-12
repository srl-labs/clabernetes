import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { listNamespaces } from "@/lib/kubernetes";
import { useQuery } from "@tanstack/react-query";
import { ChevronDown } from "lucide-react";
import type { ReactElement } from "react";

interface NamespaceSelectorProps {
  readonly namespace: string;
  readonly placeholder: string;
  readonly setNamespace: (namespace: string) => void;
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

export function NamespaceSelector(props: NamespaceSelectorProps): ReactElement {
  const { namespace, placeholder, setNamespace } = props;

  const { data, isPending, isError, error } = useQuery<string[], Error>({
    queryFn: async () => {
      const response = await listNamespaces();

      return JSON.parse(response);
    },
    queryKey: ["namespaces"],
  });

  if (isPending) {
    return (
      <div className="flex justify-end">
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
            {renderDropdownMenuTrigger(namespace, placeholder)}
            <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
          </Button>
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuContent style={{ minWidth: "30vw" }}>
        {data.map((curNamespace: string) => {
          return (
            <DropdownMenuCheckboxItem
              checked={namespace === curNamespace}
              key={curNamespace}
              onClick={() => {
                setNamespace(curNamespace);
              }}
            >
              {curNamespace}
            </DropdownMenuCheckboxItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
